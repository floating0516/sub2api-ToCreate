package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestCustomUpdateResolverResponsesEndpoint(t *testing.T) {
	for _, testCase := range []struct {
		baseURL string
		want    string
	}{
		{"https://api.example.com", "https://api.example.com/v1/responses"},
		{"https://api.example.com/v1", "https://api.example.com/v1/responses"},
		{"https://api.example.com/openai/v1", "https://api.example.com/openai/v1/responses"},
		{"https://api.example.com/v1/responses", "https://api.example.com/v1/responses"},
	} {
		endpoint, err := customUpdateResolverResponsesEndpoint(testCase.baseURL)
		require.NoError(t, err)
		require.Equal(t, testCase.want, endpoint)
	}
}

func TestCustomUpdateResolverConnectionUsesInputKeyWithoutSaving(t *testing.T) {
	const secret = "sk-current-resolver-secret"
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/responses", r.URL.Path)
		require.Equal(t, "Bearer "+secret, r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "gpt-5.6-luna", payload["model"])
		require.Equal(t, "Reply with exactly OK.", payload["input"])
		require.Equal(t, false, payload["store"])
		require.Equal(t, false, payload["stream"])
		require.EqualValues(t, 64, payload["max_output_tokens"])
		require.NotContains(t, payload, "api_key")
		reasoning, ok := payload["reasoning"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "max", reasoning["effort"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_test","object":"response","status":"completed"}`))
	}))
	defer upstream.Close()

	handler, controlDir := newCustomUpdateTestHandler(t)
	handler.resolverHTTPClient = upstream.Client()
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"base_url":"` + upstream.URL + `","model":"gpt-5.6-luna","api_key":"` + secret + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/resolver-config/test", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"ok":true`)
	require.Contains(t, recorder.Body.String(), `"model":"gpt-5.6-luna"`)
	require.Contains(t, recorder.Body.String(), `"latency_ms":`)
	require.NotContains(t, recorder.Body.String(), secret)
	entries, err := os.ReadDir(controlDir)
	require.NoError(t, err)
	require.Empty(t, entries, "connection tests must not persist resolver settings")
}

func TestCustomUpdateResolverConnectionUsesSavedKeyWhenOmitted(t *testing.T) {
	const savedKey = "sk-saved-resolver-secret"
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer "+savedKey, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"id":"resp_saved","object":"response"}`))
	}))
	defer upstream.Close()

	handler, controlDir := newCustomUpdateTestHandler(t)
	handler.resolverHTTPClient = upstream.Client()
	keyPath := filepath.Join(controlDir, customUpdateResolverAPIKeyFile)
	require.NoError(t, os.WriteFile(keyPath, []byte(savedKey+"\n"), 0600))
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"base_url":"` + upstream.URL + `/v1","model":"gpt-5.6-terra"}`)
	request := httptest.NewRequest(http.MethodPost, "/resolver-config/test", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	require.Equal(t, savedKey, strings.TrimSpace(string(keyData)))
	_, err = os.Stat(filepath.Join(controlDir, customUpdateResolverConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCustomUpdateResolverConnectionRequiresAvailableKey(t *testing.T) {
	handler, _ := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/resolver-config/test",
		bytes.NewBufferString(`{"base_url":"https://api.example.com","model":"gpt-5.6-luna"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "API key is required")
}

func TestCustomUpdateResolverConnectionMapsUpstreamAuthFailureSafely(t *testing.T) {
	const secret = "sk-rejected-resolver-secret"
	const upstreamCanary = "upstream-private-error-body"
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"` + upstreamCanary + `"}}`))
	}))
	defer upstream.Close()

	handler, _ := newCustomUpdateTestHandler(t)
	handler.resolverHTTPClient = upstream.Client()
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"base_url":"` + upstream.URL + `","model":"gpt-5.6-luna","api_key":"` + secret + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/resolver-config/test", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"upstream_status":"401"`)
	require.NotContains(t, recorder.Body.String(), secret)
	require.NotContains(t, recorder.Body.String(), upstreamCanary)
}

func TestCustomUpdateResolverConnectionDoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		redirected.Store(true)
	}))
	defer redirectTarget.Close()

	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer sk-redirect-test", r.Header.Get("Authorization"))
		http.Redirect(w, r, redirectTarget.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()

	handler, _ := newCustomUpdateTestHandler(t)
	handler.resolverHTTPClient = upstream.Client()
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"base_url":"` + upstream.URL + `","model":"gpt-5.6-luna","api_key":"sk-redirect-test"}`)
	request := httptest.NewRequest(http.MethodPost, "/resolver-config/test", body)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.False(t, redirected.Load(), "Authorization must never be forwarded through redirects")
}
