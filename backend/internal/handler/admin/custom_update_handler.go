package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	customUpdateRequestFile   = "request.json"
	customUpdateStatusFile    = "status.json"
	customUpdateHeartbeatFile = "heartbeat"
	maxCustomUpdateFileBytes  = 64 << 10
	controllerHeartbeatMaxAge = 15 * time.Second
)

type customUpdateStatus struct {
	Enabled          bool   `json:"enabled"`
	ControllerOnline bool   `json:"controller_online"`
	HeartbeatAt      string `json:"heartbeat_at,omitempty"`
	State            string `json:"state"`
	Action           string `json:"action,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
	Message          string `json:"message,omitempty"`
	Image            string `json:"image,omitempty"`
	ImageDigest      string `json:"image_digest,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	UpstreamCommit   string `json:"upstream_commit,omitempty"`
	SourceCommit     string `json:"source_commit,omitempty"`
	StartedAt        string `json:"started_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	CompletedAt      string `json:"completed_at,omitempty"`
	Error            string `json:"error,omitempty"`
	LogFile          string `json:"log_file,omitempty"`
	StagingURL       string `json:"staging_url,omitempty"`
	ProductionURL    string `json:"production_url,omitempty"`
}

type customUpdateRequest struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Image       string `json:"image,omitempty"`
	RequestedAt string `json:"requested_at"`
}

func (h *CustomBuildHandler) GetCustomUpdateStatus(c *gin.Context) {
	status, err := h.readCustomUpdateStatus()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, status)
}

func (h *CustomBuildHandler) StartCustomUpdate(c *gin.Context) {
	h.updateMu.Lock()
	defer h.updateMu.Unlock()

	status, err := h.readCustomUpdateStatus()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !status.Enabled || !status.ControllerOnline {
		response.Error(c, http.StatusServiceUnavailable, "Custom update controller is offline")
		return
	}
	if customUpdateBlocksStage(status.State) {
		response.Error(c, http.StatusConflict, "A custom update is already active or awaiting promotion")
		return
	}

	req, err := h.enqueueCustomUpdateRequest("stage", "")
	if err != nil {
		h.writeEnqueueError(c, err)
		return
	}

	response.Accepted(c, gin.H{
		"state":      "queued",
		"action":     req.Action,
		"request_id": req.ID,
		"message":    "Custom update queued for staging",
	})
}

func (h *CustomBuildHandler) PromoteCustomUpdate(c *gin.Context) {
	var body struct {
		Image string `json:"image"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	body.Image = strings.TrimSpace(body.Image)
	if body.Image == "" {
		response.BadRequest(c, "Image is required")
		return
	}

	h.updateMu.Lock()
	defer h.updateMu.Unlock()

	status, err := h.readCustomUpdateStatus()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !status.Enabled || !status.ControllerOnline {
		response.Error(c, http.StatusServiceUnavailable, "Custom update controller is offline")
		return
	}
	if status.State != "awaiting_approval" || status.Image == "" {
		response.Error(c, http.StatusConflict, "No staged custom image is awaiting promotion")
		return
	}
	if body.Image != status.Image {
		response.Error(c, http.StatusConflict, "Promotion image does not match the staged image")
		return
	}

	req, err := h.enqueueCustomUpdateRequest("promote", body.Image)
	if err != nil {
		h.writeEnqueueError(c, err)
		return
	}

	response.Accepted(c, gin.H{
		"state":      "queued",
		"action":     req.Action,
		"request_id": req.ID,
		"image":      req.Image,
		"message":    "Custom image promotion queued",
	})
}

func (h *CustomBuildHandler) readCustomUpdateStatus() (customUpdateStatus, error) {
	status := customUpdateStatus{
		State:         "disabled",
		StagingURL:    "http://127.0.0.1:18080",
		ProductionURL: "http://127.0.0.1:8080",
	}

	info, err := os.Stat(h.updateControlDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, fmt.Errorf("failed to inspect custom update control directory")
	}
	if !info.IsDir() {
		return status, fmt.Errorf("custom update control path is not a directory")
	}

	status.Enabled = true
	status.State = "idle"
	statusPath := filepath.Join(h.updateControlDir, customUpdateStatusFile)
	if err := readLimitedJSONFile(statusPath, &status); err != nil && !errors.Is(err, os.ErrNotExist) {
		return status, fmt.Errorf("failed to read custom update status")
	}
	status.Enabled = true
	if status.State == "" {
		status.State = "idle"
	}
	if status.StagingURL == "" {
		status.StagingURL = "http://127.0.0.1:18080"
	}
	if status.ProductionURL == "" {
		status.ProductionURL = "http://127.0.0.1:8080"
	}
	status.ControllerOnline = false
	status.HeartbeatAt = ""

	heartbeatPath := filepath.Join(h.updateControlDir, customUpdateHeartbeatFile)
	heartbeatInfo, heartbeatErr := os.Stat(heartbeatPath)
	if heartbeatErr == nil {
		status.HeartbeatAt = heartbeatInfo.ModTime().UTC().Format(time.RFC3339)
		status.ControllerOnline = time.Since(heartbeatInfo.ModTime()) <= controllerHeartbeatMaxAge
	} else if !errors.Is(heartbeatErr, os.ErrNotExist) {
		return status, fmt.Errorf("failed to inspect custom update controller heartbeat")
	}

	return status, nil
}

func (h *CustomBuildHandler) enqueueCustomUpdateRequest(action, image string) (*customUpdateRequest, error) {
	if _, err := os.Stat(filepath.Join(h.updateControlDir, customUpdateRequestFile)); err == nil {
		return nil, os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	id, err := newCustomUpdateRequestID()
	if err != nil {
		return nil, err
	}
	req := &customUpdateRequest{
		ID:          id,
		Action:      action,
		Image:       image,
		RequestedAt: time.Now().UTC().Format(time.RFC3339),
	}

	tempFile, err := os.CreateTemp(h.updateControlDir, ".request-*.json")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()

	// The host controller runs as the deployment user while the container runs
	// as root. The request contains only a fixed action and image reference.
	if err := tempFile.Chmod(0644); err != nil {
		_ = tempFile.Close()
		return nil, err
	}
	encoder := json.NewEncoder(tempFile)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(req); err != nil {
		_ = tempFile.Close()
		return nil, err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return nil, err
	}
	if err := tempFile.Close(); err != nil {
		return nil, err
	}

	requestPath := filepath.Join(h.updateControlDir, customUpdateRequestFile)
	if err := os.Link(tempPath, requestPath); err != nil {
		return nil, err
	}
	return req, nil
}

func (h *CustomBuildHandler) writeEnqueueError(c *gin.Context, err error) {
	if errors.Is(err, os.ErrExist) {
		response.Error(c, http.StatusConflict, "A custom update request is already queued")
		return
	}
	response.Error(c, http.StatusInternalServerError, "Failed to queue custom update request")
}

func customUpdateBlocksStage(state string) bool {
	switch state {
	case "queued", "checking", "merging", "building", "staging", "validating", "awaiting_approval", "promoting":
		return true
	default:
		return false
	}
}

func newCustomUpdateRequestID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func readLimitedJSONFile(path string, dest any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxCustomUpdateFileBytes {
		return fmt.Errorf("file exceeds %d bytes", maxCustomUpdateFileBytes)
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	return nil
}
