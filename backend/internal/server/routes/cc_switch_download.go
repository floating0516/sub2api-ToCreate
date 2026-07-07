package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ccSwitchLatestReleaseURL = "https://api.github.com/repos/farion1231/cc-switch/releases/latest"
	ccSwitchCacheTTL         = 5 * time.Minute
)

var ccSwitchAssetSuffixes = map[string][]string{
	"windows":        {"Windows.msi"},
	"windows-x86_64": {"Windows.msi"},
	"windows-arm64":  {"Windows-arm64.msi"},
	"macos":          {"macOS.dmg", "macOS.zip"},
	"mac":            {"macOS.dmg", "macOS.zip"},
	"linux":          {"Linux-x86_64.AppImage"},
	"linux-x86_64":   {"Linux-x86_64.AppImage"},
	"linux-amd64":    {"Linux-x86_64.AppImage"},
	"linux-arm64":    {"Linux-arm64.AppImage"},
}

type ccSwitchRelease struct {
	Assets []ccSwitchReleaseAsset `json:"assets"`
}

type ccSwitchReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ccSwitchDownloadHandler struct {
	latestReleaseURL string
	httpClient       *http.Client
	cacheTTL         time.Duration

	mu       sync.Mutex
	cachedAt time.Time
	assets   []ccSwitchReleaseAsset
}

func registerCCSwitchDownloadRoutes(r *gin.Engine) {
	handler := newCCSwitchDownloadHandler(ccSwitchLatestReleaseURL, &http.Client{Timeout: 10 * time.Second}, ccSwitchCacheTTL)
	r.GET("/download/cc-switch/:platform", handler.Serve)
}

func newCCSwitchDownloadHandler(latestReleaseURL string, httpClient *http.Client, cacheTTL time.Duration) *ccSwitchDownloadHandler {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &ccSwitchDownloadHandler{
		latestReleaseURL: latestReleaseURL,
		httpClient:       httpClient,
		cacheTTL:         cacheTTL,
	}
}

func (h *ccSwitchDownloadHandler) Serve(c *gin.Context) {
	platform := strings.ToLower(strings.TrimSpace(c.Param("platform")))
	suffixes, ok := ccSwitchAssetSuffixes[platform]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported cc-switch platform"})
		return
	}

	asset, err := h.findAsset(c.Request.Context(), suffixes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to resolve latest cc-switch release"})
		return
	}
	if asset == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cc-switch asset not found"})
		return
	}
	if !isTrustedCCSwitchDownloadURL(asset.BrowserDownloadURL) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid cc-switch download url"})
		return
	}

	c.Redirect(http.StatusFound, asset.BrowserDownloadURL)
}

func (h *ccSwitchDownloadHandler) findAsset(ctx context.Context, suffixes []string) (*ccSwitchReleaseAsset, error) {
	assets, err := h.latestAssets(ctx)
	if err != nil {
		return nil, err
	}
	for _, suffix := range suffixes {
		for _, asset := range assets {
			if strings.HasSuffix(asset.Name, suffix) {
				return &asset, nil
			}
		}
	}
	return nil, nil
}

func (h *ccSwitchDownloadHandler) latestAssets(ctx context.Context) ([]ccSwitchReleaseAsset, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cacheTTL > 0 && !h.cachedAt.IsZero() && time.Since(h.cachedAt) < h.cacheTTL {
		return append([]ccSwitchReleaseAsset(nil), h.assets...), nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.latestReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sub2api-cc-switch-downloader")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("github release api returned status %d", resp.StatusCode)
	}

	var release ccSwitchRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return nil, err
	}
	if release.Assets == nil {
		return nil, errors.New("github release api response missing assets")
	}

	h.cachedAt = time.Now()
	h.assets = append([]ccSwitchReleaseAsset(nil), release.Assets...)
	return append([]ccSwitchReleaseAsset(nil), release.Assets...), nil
}

func isTrustedCCSwitchDownloadURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" &&
		strings.EqualFold(parsed.Hostname(), "github.com") &&
		strings.HasPrefix(parsed.EscapedPath(), "/farion1231/cc-switch/releases/download/")
}
