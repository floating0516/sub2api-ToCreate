package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	customUpdateResolverConfigFile = "resolver-config.json"
	customUpdateResolverAPIKeyFile = "resolver-api-key"
	defaultResolverBaseURL         = "https://api.lihe.chat"
	defaultResolverModel           = "gpt-5.6-luna"
	defaultResolverReasoningEffort = "max"
	maxResolverBaseURLBytes        = 2048
	maxResolverModelBytes          = 128
	maxResolverAPIKeyBytes         = 4096
)

var customUpdateResolverModelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

type customUpdateResolverConfig struct {
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

type customUpdateResolverConfigResponse struct {
	BaseURL          string `json:"base_url"`
	Model            string `json:"model"`
	ReasoningEffort  string `json:"reasoning_effort"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	Saved            bool   `json:"saved"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	DefaultBaseURL   string `json:"default_base_url"`
	DefaultModel     string `json:"default_model"`
}

type updateCustomUpdateResolverConfigRequest struct {
	BaseURL string  `json:"base_url"`
	Model   string  `json:"model"`
	APIKey  *string `json:"api_key"`
}

func (h *CustomBuildHandler) GetCustomUpdateResolverConfig(c *gin.Context) {
	config, saved, err := h.readCustomUpdateResolverConfig()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to read conflict resolver configuration")
		return
	}
	apiKeyConfigured, err := h.customUpdateResolverAPIKeyConfigured()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to inspect conflict resolver API key")
		return
	}
	response.Success(c, newCustomUpdateResolverConfigResponse(config, saved, apiKeyConfigured))
}

func (h *CustomBuildHandler) UpdateCustomUpdateResolverConfig(c *gin.Context) {
	var body updateCustomUpdateResolverConfigRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	baseURL, err := normalizeCustomUpdateResolverBaseURL(body.BaseURL)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	model, err := normalizeCustomUpdateResolverModel(body.Model)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	apiKey, updateAPIKey, err := normalizeCustomUpdateResolverAPIKey(body.APIKey)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	h.updateMu.Lock()
	defer h.updateMu.Unlock()

	config := customUpdateResolverConfig{
		BaseURL:         baseURL,
		Model:           model,
		ReasoningEffort: defaultResolverReasoningEffort,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to encode conflict resolver configuration")
		return
	}
	configData = append(configData, '\n')

	if updateAPIKey {
		if err := h.writeCustomUpdateControlFile(
			customUpdateResolverAPIKeyFile,
			[]byte(apiKey+"\n"),
			0600,
		); err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to save conflict resolver API key")
			return
		}
	}
	if err := h.writeCustomUpdateControlFile(customUpdateResolverConfigFile, configData, 0644); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save conflict resolver configuration")
		return
	}

	apiKeyConfigured, err := h.customUpdateResolverAPIKeyConfigured()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to inspect conflict resolver API key")
		return
	}
	response.Success(c, newCustomUpdateResolverConfigResponse(config, true, apiKeyConfigured))
}

func newCustomUpdateResolverConfigResponse(
	config customUpdateResolverConfig,
	saved bool,
	apiKeyConfigured bool,
) customUpdateResolverConfigResponse {
	return customUpdateResolverConfigResponse{
		BaseURL:          config.BaseURL,
		Model:            config.Model,
		ReasoningEffort:  config.ReasoningEffort,
		APIKeyConfigured: apiKeyConfigured,
		Saved:            saved,
		UpdatedAt:        config.UpdatedAt,
		DefaultBaseURL:   defaultResolverBaseURL,
		DefaultModel:     defaultResolverModel,
	}
}

func (h *CustomBuildHandler) readCustomUpdateResolverConfig() (customUpdateResolverConfig, bool, error) {
	config := customUpdateResolverConfig{
		BaseURL:         defaultResolverBaseURL,
		Model:           defaultResolverModel,
		ReasoningEffort: defaultResolverReasoningEffort,
	}
	path := filepath.Join(h.updateControlDir, customUpdateResolverConfigFile)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, false, nil
		}
		return config, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return config, false, fmt.Errorf("resolver config path is not a regular file")
	}
	if err := readLimitedJSONFile(path, &config); err != nil {
		return config, false, err
	}

	baseURL, err := normalizeCustomUpdateResolverBaseURL(config.BaseURL)
	if err != nil {
		return config, false, err
	}
	model, err := normalizeCustomUpdateResolverModel(config.Model)
	if err != nil {
		return config, false, err
	}
	if config.ReasoningEffort == "" {
		config.ReasoningEffort = defaultResolverReasoningEffort
	}
	if config.ReasoningEffort != defaultResolverReasoningEffort {
		return config, false, fmt.Errorf("unsupported resolver reasoning effort")
	}
	config.BaseURL = baseURL
	config.Model = model
	return config, true, nil
}

func (h *CustomBuildHandler) customUpdateResolverAPIKeyConfigured() (bool, error) {
	path := filepath.Join(h.updateControlDir, customUpdateResolverAPIKeyFile)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("resolver API key path is not a regular file")
	}
	if info.Mode().Perm() != 0600 {
		return false, fmt.Errorf("resolver API key file has insecure permissions")
	}
	if info.Size() <= 0 || info.Size() > maxResolverAPIKeyBytes+1 {
		return false, fmt.Errorf("resolver API key file has an invalid size")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	key := strings.TrimSpace(string(data))
	return key != "" && len(key) <= maxResolverAPIKeyBytes, nil
}

func (h *CustomBuildHandler) writeCustomUpdateControlFile(
	name string,
	data []byte,
	mode os.FileMode,
) error {
	dirInfo, err := os.Stat(h.updateControlDir)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("custom update control path is not a directory")
	}
	dirStat, ok := dirInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("custom update control directory ownership is unavailable")
	}

	tempFile, err := os.CreateTemp(h.updateControlDir, ".resolver-config-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	tempInfo, err := tempFile.Stat()
	if err != nil {
		_ = tempFile.Close()
		return err
	}
	if tempStat, ok := tempInfo.Sys().(*syscall.Stat_t); ok &&
		(tempStat.Uid != dirStat.Uid || tempStat.Gid != dirStat.Gid) {
		if err := tempFile.Chown(int(dirStat.Uid), int(dirStat.Gid)); err != nil {
			_ = tempFile.Close()
			return err
		}
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	targetPath := filepath.Join(h.updateControlDir, name)
	if targetInfo, err := os.Lstat(targetPath); err == nil && targetInfo.IsDir() {
		return fmt.Errorf("custom update control target is a directory")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return err
	}
	dir, err := os.Open(h.updateControlDir)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return dir.Sync()
}

func normalizeCustomUpdateResolverBaseURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxResolverBaseURLBytes {
		return "", fmt.Errorf("Base URL is required and must not exceed %d characters", maxResolverBaseURLBytes)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("Base URL must be a valid HTTPS URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", fmt.Errorf("Base URL must not contain credentials, query parameters, or fragments")
	}
	return strings.TrimRight(value, "/"), nil
}

func normalizeCustomUpdateResolverModel(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxResolverModelBytes || !customUpdateResolverModelPattern.MatchString(value) {
		return "", fmt.Errorf("Model must use letters, numbers, dots, underscores, colons, slashes, or hyphens")
	}
	return value, nil
}

func normalizeCustomUpdateResolverAPIKey(value *string) (string, bool, error) {
	if value == nil {
		return "", false, nil
	}
	if strings.ContainsAny(*value, "\r\n") {
		return "", false, fmt.Errorf("API key must be a single line")
	}
	key := strings.TrimSpace(*value)
	if key == "" || len(key) > maxResolverAPIKeyBytes {
		return "", false, fmt.Errorf("API key is required and must not exceed %d characters", maxResolverAPIKeyBytes)
	}
	return key, true, nil
}
