package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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
	maxResolverTestResponseBytes   = 1 << 20
	defaultResolverTestTimeout     = 60 * time.Second
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

type customUpdateResolverTestResponse struct {
	OK        bool   `json:"ok"`
	Model     string `json:"model"`
	LatencyMS int64  `json:"latency_ms"`
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

func (h *CustomBuildHandler) TestCustomUpdateResolverConfig(c *gin.Context) {
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
	apiKey, provided, err := normalizeCustomUpdateResolverAPIKey(body.APIKey)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if !provided {
		apiKey, err = h.readCustomUpdateResolverAPIKey()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to read conflict resolver API key")
			return
		}
		if apiKey == "" {
			response.BadRequest(c, "An API key is required to test the connection")
			return
		}
	}

	endpoint, err := customUpdateResolverResponsesEndpoint(baseURL)
	if err != nil {
		response.BadRequest(c, "Failed to construct the Responses API endpoint")
		return
	}
	payload, err := json.Marshal(gin.H{
		"model":             model,
		"input":             "Reply with exactly OK.",
		"store":             false,
		"stream":            false,
		"reasoning":         gin.H{"effort": defaultResolverReasoningEffort},
		"max_output_tokens": 64,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to prepare the connection test")
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		response.BadRequest(c, "Failed to construct the Responses API request")
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	startedAt := time.Now()
	upstreamResponse, err := h.customUpdateResolverTestHTTPClient().Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || resolverTestErrorTimedOut(err) {
			response.Error(c, http.StatusGatewayTimeout, "The upstream connection test timed out")
			return
		}
		response.Error(c, http.StatusBadGateway, "Unable to reach the upstream service")
		return
	}
	defer func() { _ = upstreamResponse.Body.Close() }()

	upstreamBody, err := io.ReadAll(io.LimitReader(upstreamResponse.Body, maxResolverTestResponseBytes+1))
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Failed to read the upstream response")
		return
	}
	if len(upstreamBody) > maxResolverTestResponseBytes {
		response.Error(c, http.StatusBadGateway, "The upstream response was unexpectedly large")
		return
	}
	if upstreamResponse.StatusCode < http.StatusOK || upstreamResponse.StatusCode >= http.StatusMultipleChoices {
		writeCustomUpdateResolverUpstreamError(c, upstreamResponse.StatusCode)
		return
	}
	if !validCustomUpdateResolverTestResponse(upstreamBody) {
		response.Error(c, http.StatusBadGateway, "The upstream service returned an invalid Responses API response")
		return
	}

	latencyMS := time.Since(startedAt).Milliseconds()
	if latencyMS < 1 {
		latencyMS = 1
	}
	response.Success(c, customUpdateResolverTestResponse{
		OK:        true,
		Model:     model,
		LatencyMS: latencyMS,
	})
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
	key, err := h.readCustomUpdateResolverAPIKey()
	return key != "", err
}

func (h *CustomBuildHandler) readCustomUpdateResolverAPIKey() (string, error) {
	path := filepath.Join(h.updateControlDir, customUpdateResolverAPIKeyFile)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("resolver API key path is not a regular file")
	}
	if info.Mode().Perm() != 0600 {
		return "", fmt.Errorf("resolver API key file has insecure permissions")
	}
	if info.Size() <= 0 || info.Size() > maxResolverAPIKeyBytes+1 {
		return "", fmt.Errorf("resolver API key file has an invalid size")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(data))
	if key == "" || len(key) > maxResolverAPIKeyBytes || strings.ContainsAny(key, "\r\n") {
		return "", fmt.Errorf("resolver API key file contains an invalid key")
	}
	return key, nil
}

func (h *CustomBuildHandler) customUpdateResolverTestHTTPClient() *http.Client {
	baseClient := h.resolverHTTPClient
	if baseClient == nil {
		baseClient = http.DefaultClient
	}
	client := *baseClient
	if client.Timeout <= 0 {
		client.Timeout = defaultResolverTestTimeout
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

func customUpdateResolverResponsesEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/v1/responses"):
	case strings.HasSuffix(path, "/v1"):
		path += "/responses"
	default:
		path += "/v1/responses"
	}
	if path == "" {
		path = "/v1/responses"
	}
	parsed.Path = path
	parsed.RawPath = ""
	return parsed.String(), nil
}

func resolverTestErrorTimedOut(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func validCustomUpdateResolverTestResponse(data []byte) bool {
	var result struct {
		ID     string          `json:"id"`
		Object string          `json:"object"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return false
	}
	if len(result.Error) > 0 && string(result.Error) != "null" {
		return false
	}
	return result.ID != "" || result.Object == "response"
}

func writeCustomUpdateResolverUpstreamError(c *gin.Context, upstreamStatus int) {
	status := http.StatusUnprocessableEntity
	message := "The upstream service rejected the connection test"
	switch upstreamStatus {
	case http.StatusUnauthorized, http.StatusForbidden:
		message = "The API key was rejected by the upstream service"
	case http.StatusNotFound:
		message = "The Responses API endpoint or model was not found"
	case http.StatusTooManyRequests:
		message = "The upstream rate limit was reached; try again later"
	default:
		if upstreamStatus >= http.StatusInternalServerError {
			status = http.StatusBadGateway
			message = "The upstream service is temporarily unavailable"
		}
	}
	response.ErrorWithDetails(c, status, message, "upstream_test_failed", map[string]string{
		"upstream_status": fmt.Sprintf("%d", upstreamStatus),
	})
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
		return "", fmt.Errorf("base URL is required and must not exceed %d characters", maxResolverBaseURLBytes)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("base URL must be a valid HTTPS URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", fmt.Errorf("base URL must not contain credentials, query parameters, or fragments")
	}
	return strings.TrimRight(value, "/"), nil
}

func normalizeCustomUpdateResolverModel(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxResolverModelBytes || !customUpdateResolverModelPattern.MatchString(value) {
		return "", fmt.Errorf("model must use letters, numbers, dots, underscores, colons, slashes, or hyphens")
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
