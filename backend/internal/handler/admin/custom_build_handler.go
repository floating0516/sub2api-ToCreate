package admin

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	customBuildNotesPathEnv     = "CUSTOM_BUILD_NOTES_PATH"
	defaultCustomBuildNotesPath = "/app/custom/CUSTOM_BUILD_NOTES.md"
	maxCustomBuildNotesBytes    = 1 << 20
)

type CustomBuildHandler struct {
	notesPath string
}

func NewCustomBuildHandler() *CustomBuildHandler {
	path := strings.TrimSpace(os.Getenv(customBuildNotesPathEnv))
	if path == "" {
		path = defaultCustomBuildNotesPath
	}
	return &CustomBuildHandler{notesPath: path}
}

func (h *CustomBuildHandler) GetNotes(c *gin.Context) {
	info, err := os.Stat(h.notesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			response.NotFound(c, "Custom build notes file not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to read custom build notes metadata")
		return
	}
	if info.IsDir() {
		response.Error(c, http.StatusInternalServerError, "Custom build notes path is a directory")
		return
	}
	if info.Size() > maxCustomBuildNotesBytes {
		response.Error(c, http.StatusRequestEntityTooLarge, "Custom build notes file is too large")
		return
	}

	content, err := os.ReadFile(h.notesPath)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to read custom build notes")
		return
	}

	response.Success(c, gin.H{
		"content":    string(content),
		"path":       h.notesPath,
		"updated_at": info.ModTime().Format(time.RFC3339),
	})
}
