package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	CodexGeneratedImageRoutePattern = "/v1/generated-images/:name"

	codexGeneratedImageRoutePrefix    = "/v1/generated-images/"
	codexGeneratedImageDirectory      = "codex-generated-images"
	codexGeneratedImageRandomBytes    = 24
	codexGeneratedImageMaxDecodedSize = int64(32 << 20)
	codexGeneratedImageTTL            = 30 * 24 * time.Hour
	codexGeneratedImageCleanupEvery   = time.Hour
)

type codexGeneratedImageArtifact struct {
	Name        string
	ContentType string
	Size        int64
}

type codexGeneratedImageStore struct {
	dir             string
	maxDecodedBytes int64
	ttl             time.Duration
	cleanupEvery    time.Duration
	now             func() time.Time
	random          io.Reader

	mu          sync.Mutex
	lastCleanup time.Time
}

func newCodexGeneratedImageStore(cfg *config.Config) *codexGeneratedImageStore {
	if cfg == nil {
		return nil
	}
	dataDir := strings.TrimSpace(cfg.Pricing.DataDir)
	if dataDir == "" {
		return nil
	}
	return &codexGeneratedImageStore{
		dir:             filepath.Join(dataDir, codexGeneratedImageDirectory),
		maxDecodedBytes: codexGeneratedImageMaxDecodedSize,
		ttl:             codexGeneratedImageTTL,
		cleanupEvery:    codexGeneratedImageCleanupEvery,
		now:             time.Now,
		random:          rand.Reader,
	}
}

func (s *OpenAIGatewayService) getCodexGeneratedImageStore() *codexGeneratedImageStore {
	if s == nil {
		return nil
	}
	s.codexImageArtifactStoreOnce.Do(func() {
		if s.codexImageArtifactStore == nil {
			s.codexImageArtifactStore = newCodexGeneratedImageStore(s.cfg)
		}
	})
	return s.codexImageArtifactStore
}

func (s *codexGeneratedImageStore) SaveBase64(result, declaredFormat string) (codexGeneratedImageArtifact, error) {
	if s == nil || strings.TrimSpace(s.dir) == "" {
		return codexGeneratedImageArtifact{}, errors.New("generated image storage is unavailable")
	}
	encoded, dataMIME, err := parseCodexGeneratedImageBase64(result)
	if err != nil {
		return codexGeneratedImageArtifact{}, err
	}
	maxDecodedBytes := s.maxDecodedBytes
	if maxDecodedBytes <= 0 {
		maxDecodedBytes = codexGeneratedImageMaxDecodedSize
	}
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > maxDecodedBytes {
		return codexGeneratedImageArtifact{}, fmt.Errorf("generated image exceeds %d decoded bytes", maxDecodedBytes)
	}

	decoded, err := decodeCodexGeneratedImageBase64(encoded)
	if err != nil {
		return codexGeneratedImageArtifact{}, fmt.Errorf("decode generated image: %w", err)
	}
	if len(decoded) == 0 {
		return codexGeneratedImageArtifact{}, errors.New("generated image is empty")
	}
	if int64(len(decoded)) > maxDecodedBytes {
		return codexGeneratedImageArtifact{}, fmt.Errorf("generated image exceeds %d decoded bytes", maxDecodedBytes)
	}

	contentType := http.DetectContentType(decoded)
	extension, canonicalFormat, ok := codexGeneratedImageType(contentType)
	if !ok {
		return codexGeneratedImageArtifact{}, fmt.Errorf("unsupported generated image MIME type %q", contentType)
	}
	if err := validateCodexGeneratedImageDeclaredType(declaredFormat, canonicalFormat); err != nil {
		return codexGeneratedImageArtifact{}, err
	}
	if err := validateCodexGeneratedImageDeclaredType(dataMIME, canonicalFormat); err != nil {
		return codexGeneratedImageArtifact{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return codexGeneratedImageArtifact{}, fmt.Errorf("create generated image directory: %w", err)
	}
	s.cleanupExpiredLocked()

	randomSource := s.random
	if randomSource == nil {
		randomSource = rand.Reader
	}
	randomBytes := make([]byte, codexGeneratedImageRandomBytes)
	if _, err := io.ReadFull(randomSource, randomBytes); err != nil {
		return codexGeneratedImageArtifact{}, fmt.Errorf("create generated image name: %w", err)
	}
	name := hex.EncodeToString(randomBytes) + extension
	path := filepath.Join(s.dir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return codexGeneratedImageArtifact{}, fmt.Errorf("create generated image file: %w", err)
	}
	var writeErr error
	if _, err := file.Write(decoded); err != nil {
		writeErr = err
	}
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = os.Remove(path)
		return codexGeneratedImageArtifact{}, fmt.Errorf("write generated image file: %w", writeErr)
	}

	return codexGeneratedImageArtifact{
		Name:        name,
		ContentType: contentType,
		Size:        int64(len(decoded)),
	}, nil
}

func parseCodexGeneratedImageBase64(result string) (encoded string, dataMIME string, err error) {
	value := strings.TrimSpace(result)
	if value == "" {
		return "", "", errors.New("generated image result is empty")
	}
	if !strings.HasPrefix(strings.ToLower(value), "data:") {
		return value, "", nil
	}

	comma := strings.IndexByte(value, ',')
	if comma <= len("data:") {
		return "", "", errors.New("invalid generated image data URI")
	}
	metadata := value[len("data:"):comma]
	parts := strings.Split(metadata, ";")
	if len(parts) == 0 || !strings.EqualFold(strings.TrimSpace(parts[len(parts)-1]), "base64") {
		return "", "", errors.New("generated image data URI is not base64 encoded")
	}
	dataMIME = strings.TrimSpace(parts[0])
	return strings.TrimSpace(value[comma+1:]), dataMIME, nil
}

func decodeCodexGeneratedImageBase64(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(encoded)
}

func codexGeneratedImageType(contentType string) (extension string, canonicalFormat string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])) {
	case "image/png":
		return ".png", "png", true
	case "image/jpeg":
		return ".jpg", "jpeg", true
	case "image/webp":
		return ".webp", "webp", true
	default:
		return "", "", false
	}
}

func validateCodexGeneratedImageDeclaredType(declared, actual string) error {
	declared = strings.ToLower(strings.TrimSpace(strings.Split(declared, ";")[0]))
	switch declared {
	case "":
		return nil
	case "jpg", "jpeg", "image/jpg", "image/jpeg":
		declared = "jpeg"
	case "png", "image/png":
		declared = "png"
	case "webp", "image/webp":
		declared = "webp"
	default:
		return fmt.Errorf("unsupported declared generated image type %q", declared)
	}
	if declared != actual {
		return fmt.Errorf("generated image type mismatch: declared %s, detected %s", declared, actual)
	}
	return nil
}

func (s *codexGeneratedImageStore) cleanupExpiredLocked() {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	cleanupEvery := s.cleanupEvery
	if cleanupEvery <= 0 {
		cleanupEvery = codexGeneratedImageCleanupEvery
	}
	if !s.lastCleanup.IsZero() && now.Sub(s.lastCleanup) < cleanupEvery {
		return
	}
	s.lastCleanup = now

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		logger.LegacyPrintf("service.openai_gateway", "Codex generated image cleanup failed: %v", err)
		return
	}
	ttl := s.ttl
	if ttl <= 0 {
		ttl = codexGeneratedImageTTL
	}
	cutoff := now.Add(-ttl)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.LegacyPrintf("service.openai_gateway", "Codex generated image cleanup remove failed: %v", err)
		}
	}
}

func (s *codexGeneratedImageStore) Open(name string) (*os.File, os.FileInfo, string, error) {
	if s == nil || !validCodexGeneratedImageName(name) {
		return nil, nil, "", os.ErrNotExist
	}
	path := filepath.Join(s.dir, name)
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, "", err
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, nil, "", os.ErrNotExist
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	ttl := s.ttl
	if ttl <= 0 {
		ttl = codexGeneratedImageTTL
	}
	if info.ModTime().Before(now.Add(-ttl)) {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, nil, "", os.ErrNotExist
	}
	contentType, ok := codexGeneratedImageContentTypeFromName(name)
	if !ok {
		_ = file.Close()
		return nil, nil, "", os.ErrNotExist
	}
	return file, info, contentType, nil
}

func validCodexGeneratedImageName(name string) bool {
	if filepath.Base(name) != name {
		return false
	}
	extension := strings.ToLower(filepath.Ext(name))
	if extension != ".png" && extension != ".jpg" && extension != ".webp" {
		return false
	}
	stem := strings.TrimSuffix(name, extension)
	if len(stem) != codexGeneratedImageRandomBytes*2 {
		return false
	}
	decoded, err := hex.DecodeString(stem)
	return err == nil && len(decoded) == codexGeneratedImageRandomBytes
}

func codexGeneratedImageContentTypeFromName(name string) (string, bool) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png", true
	case ".jpg":
		return "image/jpeg", true
	case ".webp":
		return "image/webp", true
	default:
		return "", false
	}
}

func (s *OpenAIGatewayService) ServeCodexGeneratedImage(c *gin.Context) {
	if c == nil {
		return
	}
	store := s.getCodexGeneratedImageStore()
	if store == nil {
		c.Status(http.StatusNotFound)
		return
	}
	name := c.Param("name")
	file, info, contentType, err := store.Open(name)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.LegacyPrintf("service.openai_gateway", "Open Codex generated image failed: %v", err)
		}
		c.Status(http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, name))
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, name, info.ModTime(), file)
}
