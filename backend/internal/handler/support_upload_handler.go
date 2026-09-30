package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/setup"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const supportMaxUploadBytes int64 = 10 << 20

// SetUploadsDir reuses the console's persistent data volume, outside public pages.
func (h *SupportHandler) SetUploadsDir(dataDir string) {
	if strings.TrimSpace(dataDir) == "" {
		dataDir = setup.GetDataDir()
	}
	h.uploadsDir = filepath.Join(dataDir, "support", "uploads")
}

func (h *SupportHandler) Upload(c *gin.Context)              { h.upload(c, "/uploads") }
func (h *SupportHandler) AdminUpload(c *gin.Context)         { h.upload(c, "/admin/uploads") }
func (h *SupportHandler) DownloadUpload(c *gin.Context)      { h.downloadUpload(c, "/uploads") }
func (h *SupportHandler) AdminDownloadUpload(c *gin.Context) { h.downloadUpload(c, "/admin/uploads") }

func (h *SupportHandler) upload(c *gin.Context, path string) {
	if !h.supportReady(c, path) {
		return
	}
	if h.uploadsDir == "" {
		response.Error(c, http.StatusServiceUnavailable, "support upload storage unavailable")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, supportMaxUploadBytes+64*1024)
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		response.BadRequest(c, "invalid or oversized upload")
		return
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	files := c.Request.MultipartForm.File["file"]
	if len(files) != 1 || len(c.Request.MultipartForm.File) != 1 {
		response.BadRequest(c, "one file is required")
		return
	}
	file, err := files[0].Open()
	if err != nil {
		response.BadRequest(c, "invalid upload")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, supportMaxUploadBytes+1))
	if err != nil || len(data) == 0 || int64(len(data)) > supportMaxUploadBytes {
		response.BadRequest(c, "file must be between 1 byte and 10MB")
		return
	}
	filename := filepath.Base(strings.ReplaceAll(files[0].Filename, "\\", "/"))
	if len(filename) > 255 || strings.ContainsAny(filename, "\r\n") {
		response.BadRequest(c, "invalid filename")
		return
	}
	contentType := strings.Split(http.DetectContentType(data), ";")[0]
	if !validSupportUpload(filename, contentType, data) {
		response.BadRequest(c, "only png/jpeg/webp/pdf/txt files are allowed")
		return
	}
	metadata := map[string]any{"id": uuid.NewString(), "filename": filename, "size": len(data), "content_type": contentType}
	if ticketID := c.PostForm("ticket_id"); ticketID != "" {
		if _, err := uuid.Parse(ticketID); err != nil {
			response.BadRequest(c, "invalid ticket_id")
			return
		}
		metadata["ticket_id"] = ticketID
	}
	if err := os.MkdirAll(h.uploadsDir, 0700); err != nil {
		response.Error(c, http.StatusServiceUnavailable, "support upload storage unavailable")
		return
	}
	storedPath := filepath.Join(h.uploadsDir, metadata["id"].(string))
	stored, err := os.OpenFile(storedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "support upload storage unavailable")
		return
	}
	_, writeErr := stored.Write(data)
	closeErr := stored.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(storedPath)
		response.Error(c, http.StatusServiceUnavailable, "support upload storage unavailable")
		return
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		_ = os.Remove(storedPath)
		response.Error(c, http.StatusInternalServerError, "invalid upload metadata")
		return
	}
	payload, ok := h.requestJSON(c, http.MethodPost, path, body)
	if !ok {
		_ = os.Remove(storedPath)
		return
	}
	response.Success(c, payload)
}

func validSupportUpload(filename, contentType string, data []byte) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch contentType {
	case "image/png":
		return ext == ".png"
	case "image/jpeg":
		return ext == ".jpg" || ext == ".jpeg"
	case "image/webp":
		return ext == ".webp"
	case "application/pdf":
		return ext == ".pdf"
	case "text/plain":
		return ext == ".txt" && utf8.Valid(data) && !strings.ContainsRune(string(data), '\x00')
	}
	return false
}

func (h *SupportHandler) downloadUpload(c *gin.Context, path string) {
	if !h.validPortalID(c, path) {
		return
	}
	if h.uploadsDir == "" {
		response.Error(c, http.StatusServiceUnavailable, "support upload storage unavailable")
		return
	}
	payload, ok := h.requestJSON(c, http.MethodGet, path+"/"+c.Param("id"), nil)
	if !ok {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "invalid attachment metadata")
		return
	}
	var metadata struct {
		StorageKey  string `json:"storage_key"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := json.Unmarshal(raw, &metadata); err != nil || metadata.StorageKey != c.Param("id") || metadata.Size <= 0 || metadata.Size > supportMaxUploadBytes {
		response.Error(c, http.StatusBadGateway, "invalid attachment metadata")
		return
	}
	file, err := os.Open(filepath.Join(h.uploadsDir, metadata.StorageKey))
	if err != nil {
		response.NotFound(c, "attachment file not found")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, supportMaxUploadBytes+1))
	if err != nil || int64(len(data)) != metadata.Size || !validSupportUpload(metadata.Filename, metadata.ContentType, data) {
		response.Error(c, http.StatusBadGateway, "invalid attachment file")
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": metadata.Filename}))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Security-Policy", "sandbox")
	c.Data(http.StatusOK, metadata.ContentType, data)
}
