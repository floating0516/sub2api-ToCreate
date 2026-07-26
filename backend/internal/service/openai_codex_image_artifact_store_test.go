package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const codexTestPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func newCodexGeneratedImageTestStore(t *testing.T) (*codexGeneratedImageStore, *config.Config) {
	t.Helper()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = t.TempDir()
	store := newCodexGeneratedImageStore(cfg)
	require.NotNil(t, store)
	return store, cfg
}

func TestCodexGeneratedImageStoreSaveOpenAndServe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, cfg := newCodexGeneratedImageTestStore(t)
	artifact, err := store.SaveBase64(codexTestPNGBase64, "png")
	require.NoError(t, err)
	require.True(t, validCodexGeneratedImageName(artifact.Name))
	require.Equal(t, "image/png", artifact.ContentType)

	expected, err := base64.StdEncoding.DecodeString(codexTestPNGBase64)
	require.NoError(t, err)
	require.Equal(t, int64(len(expected)), artifact.Size)

	file, info, contentType, err := store.Open(artifact.Name)
	require.NoError(t, err)
	require.Equal(t, "image/png", contentType)
	require.Equal(t, int64(len(expected)), info.Size())
	actual, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, expected, actual)

	svc := &OpenAIGatewayService{
		cfg:                     cfg,
		codexImageArtifactStore: store,
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "name", Value: artifact.Name}}
	c.Request = httptest.NewRequest(http.MethodGet, CodexGeneratedImageRoutePattern, nil)

	svc.ServeCodexGeneratedImage(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Equal(t, expected, recorder.Body.Bytes())
}

func TestCodexGeneratedImageStoreRejectsInvalidPathsAndExpiredFiles(t *testing.T) {
	store, _ := newCodexGeneratedImageTestStore(t)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	artifact, err := store.SaveBase64(codexTestPNGBase64, "image/png")
	require.NoError(t, err)

	for _, name := range []string{
		"../" + artifact.Name,
		artifact.Name + ".png",
		"not-a-random-image.png",
	} {
		file, _, _, openErr := store.Open(name)
		require.ErrorIs(t, openErr, os.ErrNotExist)
		require.Nil(t, file)
	}

	path := filepath.Join(store.dir, artifact.Name)
	expiredAt := now.Add(-codexGeneratedImageTTL - time.Hour)
	require.NoError(t, os.Chtimes(path, expiredAt, expiredAt))

	file, _, _, err := store.Open(artifact.Name)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Nil(t, file)
	_, statErr := os.Stat(path)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestCodexGeneratedImageStoreValidatesSizeAndMIME(t *testing.T) {
	store, _ := newCodexGeneratedImageTestStore(t)
	store.maxDecodedBytes = 4
	_, err := store.SaveBase64(codexTestPNGBase64, "png")
	require.ErrorContains(t, err, "exceeds")

	store.maxDecodedBytes = codexGeneratedImageMaxDecodedSize
	_, err = store.SaveBase64(base64.StdEncoding.EncodeToString([]byte("not an image")), "png")
	require.ErrorContains(t, err, "unsupported generated image MIME type")

	_, err = store.SaveBase64(codexTestPNGBase64, "jpeg")
	require.ErrorContains(t, err, "type mismatch")

	_, err = store.SaveBase64("not-base64", "png")
	require.Error(t, err)
	require.False(t, errors.Is(err, os.ErrNotExist))
}

func TestCodexGeneratedImagePublicOriginUsesForwardedHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "http://api.example.test/v1/responses", bytes.NewReader(nil))
	c.Request.Host = "api.example.test"
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	origin, err := codexGeneratedImagePublicOrigin(c)
	require.NoError(t, err)
	require.Equal(t, "https://api.example.test", origin)
}
