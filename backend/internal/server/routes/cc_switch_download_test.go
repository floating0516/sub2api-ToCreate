package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCCSwitchDownloadHandlerRedirectsToLatestAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"assets": [
				{"name": "CC-Switch-v3.16.5-Windows-arm64.msi", "browser_download_url": "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-Windows-arm64.msi"},
				{"name": "CC-Switch-v3.16.5-Windows.msi", "browser_download_url": "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-Windows.msi"},
				{"name": "CC-Switch-v3.16.5-macOS.dmg", "browser_download_url": "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-macOS.dmg"},
				{"name": "CC-Switch-v3.16.5-Linux-x86_64.AppImage", "browser_download_url": "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-Linux-x86_64.AppImage"}
			]
		}`))
	}))
	defer upstream.Close()

	router := gin.New()
	handler := newCCSwitchDownloadHandler(upstream.URL, upstream.Client(), time.Minute)
	router.GET("/download/cc-switch/:platform", handler.Serve)

	tests := []struct {
		platform string
		location string
	}{
		{
			platform: "windows",
			location: "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-Windows.msi",
		},
		{
			platform: "windows-arm64",
			location: "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-Windows-arm64.msi",
		},
		{
			platform: "macos",
			location: "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-macOS.dmg",
		},
		{
			platform: "linux-x86_64",
			location: "https://github.com/farion1231/cc-switch/releases/download/v3.16.5/CC-Switch-v3.16.5-Linux-x86_64.AppImage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/download/cc-switch/"+tt.platform, nil)

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusFound, w.Code)
			require.Equal(t, tt.location, w.Header().Get("Location"))
		})
	}
}

func TestCCSwitchDownloadHandlerRejectsUnsupportedPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := newCCSwitchDownloadHandler("http://127.0.0.1:1", nil, time.Minute)
	router.GET("/download/cc-switch/:platform", handler.Serve)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/cc-switch/solaris", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCCSwitchDownloadHandlerRejectsUntrustedAssetURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"assets": [
				{"name": "CC-Switch-v3.16.5-Windows.msi", "browser_download_url": "https://example.com/CC-Switch-v3.16.5-Windows.msi"}
			]
		}`))
	}))
	defer upstream.Close()

	router := gin.New()
	handler := newCCSwitchDownloadHandler(upstream.URL, upstream.Client(), time.Minute)
	router.GET("/download/cc-switch/:platform", handler.Serve)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download/cc-switch/windows", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
}
