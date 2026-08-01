package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCustomUpdateResolverConfigReturnsDefaultsWithoutSecret(t *testing.T) {
	handler, _ := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resolver-config", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"base_url":"https://api.lihe.chat"`)
	require.Contains(t, recorder.Body.String(), `"model":"gpt-5.6-luna"`)
	require.Contains(t, recorder.Body.String(), `"api_key_configured":false`)
	require.Contains(t, recorder.Body.String(), `"saved":false`)
	require.Contains(t, recorder.Body.String(), `"default_model":"gpt-5.6-luna"`)
	require.NotContains(t, recorder.Body.String(), `"api_key":`)
}

func TestUpdateCustomUpdateResolverConfigPersistsKeySecurely(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)
	const secret = "sk-test-resolver-secret"

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
  "base_url":"https://gateway.example.com/v1/",
  "model":"gpt-5.6-terra",
  "api_key":"` + secret + `"
}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/resolver-config", body))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), secret)
	require.Contains(t, recorder.Body.String(), `"base_url":"https://gateway.example.com/v1"`)
	require.Contains(t, recorder.Body.String(), `"api_key_configured":true`)

	configData, err := os.ReadFile(filepath.Join(controlDir, customUpdateResolverConfigFile))
	require.NoError(t, err)
	var config customUpdateResolverConfig
	require.NoError(t, json.Unmarshal(configData, &config))
	require.Equal(t, "https://gateway.example.com/v1", config.BaseURL)
	require.Equal(t, "gpt-5.6-terra", config.Model)
	require.Equal(t, "max", config.ReasoningEffort)

	keyPath := filepath.Join(controlDir, customUpdateResolverAPIKeyFile)
	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	require.Equal(t, secret, strings.TrimSpace(string(keyData)))
	keyInfo, err := os.Stat(keyPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), keyInfo.Mode().Perm())

	configInfo, err := os.Stat(filepath.Join(controlDir, customUpdateResolverConfigFile))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), configInfo.Mode().Perm())
}

func TestUpdateCustomUpdateResolverConfigLeavesSavedKeyWhenOmitted(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)
	keyPath := filepath.Join(controlDir, customUpdateResolverAPIKeyFile)
	require.NoError(t, os.WriteFile(keyPath, []byte("saved-key\n"), 0600))

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
  "base_url":"https://api.lihe.chat",
  "model":"gpt-5.6-luna"
}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/resolver-config", body))

	require.Equal(t, http.StatusOK, recorder.Code)
	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	require.Equal(t, "saved-key", strings.TrimSpace(string(keyData)))
}

func TestUpdateCustomUpdateResolverConfigRejectsInvalidValues(t *testing.T) {
	handler, _ := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)

	for _, body := range []string{
		`{"base_url":"http://api.example.com","model":"gpt-5.6-luna"}`,
		`{"base_url":"https://user:password@api.example.com","model":"gpt-5.6-luna"}`,
		`{"base_url":"https://api.example.com?v=1","model":"gpt-5.6-luna"}`,
		`{"base_url":"https://api.example.com","model":"bad model"}`,
		`{"base_url":"https://api.example.com","model":"gpt-5.6-luna","api_key":"line1\nline2"}`,
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(
			recorder,
			httptest.NewRequest(http.MethodPut, "/resolver-config", bytes.NewBufferString(body)),
		)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestGetCustomUpdateResolverConfigRejectsSymlink(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)
	target := filepath.Join(t.TempDir(), "resolver-config.json")
	require.NoError(t, os.WriteFile(target, []byte(`{
  "base_url":"https://api.example.com",
  "model":"gpt-5.6-luna",
  "reasoning_effort":"max"
}`), 0644))
	require.NoError(t, os.Symlink(target, filepath.Join(controlDir, customUpdateResolverConfigFile)))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resolver-config", nil))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}
