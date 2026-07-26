package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func runCodexImageArtifactStreamTest(
	t *testing.T,
	input string,
	fallback *codexImageArtifactFallback,
) string {
	t.Helper()
	source := io.NopCloser(strings.NewReader(input))
	reader, writer := io.Pipe()
	go transformCodexImageArtifactStream(source, writer, fallback, defaultMaxLineSize)
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	return string(output)
}

func codexImageArtifactTestSSE(imageItem string) string {
	return strings.Join([]string{
		"event: response.output_item.done",
		`data: {"type":"response.output_item.done","sequence_number":5,"output_index":0,"item":` + imageItem + `}`,
		"",
		"event: response.output_item.done",
		`data: {"type":"response.output_item.done","sequence_number":6,"output_index":0,"item":` + imageItem + `}`,
		"",
		"event: response.completed",
		`data: {"type":"response.completed","sequence_number":7,"response":{"id":"resp_image","object":"response","status":"completed","model":"gpt-5.4","output":[],"usage":{"input_tokens":1,"output_tokens":2}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
}

func collectCodexImageArtifactPayloads(body string) []string {
	payloads := make([]string, 0)
	forEachOpenAISSEDataPayload(body, func(data []byte) {
		if string(data) != "[DONE]" {
			payloads = append(payloads, string(data))
		}
	})
	return payloads
}

func findCodexImageArtifactPayload(t *testing.T, payloads []string, eventType string, itemType string) string {
	t.Helper()
	for _, payload := range payloads {
		if gjson.Get(payload, "type").String() != eventType {
			continue
		}
		if itemType != "" && gjson.Get(payload, "item.type").String() != itemType {
			continue
		}
		return payload
	}
	t.Fatalf("missing event type=%s item_type=%s", eventType, itemType)
	return ""
}

func TestCodexImageArtifactStreamInjectsMarkdownMessageAndDeduplicates(t *testing.T) {
	store, _ := newCodexGeneratedImageTestStore(t)
	fallback := newCodexImageArtifactFallback(store, "https://api.example.test")
	imageItem := fmt.Sprintf(
		`{"id":"ig_test","type":"image_generation_call","status":"generating","result":%s,"output_format":"png"}`,
		strconv.Quote(codexTestPNGBase64),
	)

	output := runCodexImageArtifactStreamTest(t, codexImageArtifactTestSSE(imageItem), fallback)
	payloads := collectCodexImageArtifactPayloads(output)

	files, err := os.ReadDir(store.dir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.True(t, validCodexGeneratedImageName(files[0].Name()))

	delta := findCodexImageArtifactPayload(t, payloads, "response.output_text.delta", "")
	text := gjson.Get(delta, "delta").String()
	require.Contains(t, text, "![Generated image](https://api.example.test/v1/generated-images/")
	require.Contains(t, text, "\n\nDownload: https://api.example.test/v1/generated-images/")
	require.Equal(t, int64(9), gjson.Get(delta, "sequence_number").Int())
	require.Equal(t, int64(1), gjson.Get(delta, "output_index").Int())

	added := findCodexImageArtifactPayload(t, payloads, "response.output_item.added", "message")
	require.Equal(t, int64(7), gjson.Get(added, "sequence_number").Int())
	done := findCodexImageArtifactPayload(t, payloads, "response.output_item.done", "message")
	require.Equal(t, int64(12), gjson.Get(done, "sequence_number").Int())

	terminal := findCodexImageArtifactPayload(t, payloads, "response.completed", "")
	require.Equal(t, int64(13), gjson.Get(terminal, "sequence_number").Int())
	require.Len(t, gjson.Get(terminal, "response.output").Array(), 2)
	require.Equal(t, "image_generation_call", gjson.Get(terminal, "response.output.0.type").String())
	require.Equal(t, "message", gjson.Get(terminal, "response.output.1.type").String())
	require.Equal(t, text, gjson.Get(terminal, "response.output.1.content.0.text").String())

	messageDoneCount := 0
	for _, payload := range payloads {
		if gjson.Get(payload, "type").String() == "response.output_item.done" &&
			gjson.Get(payload, "item.type").String() == "message" {
			messageDoneCount++
		}
	}
	require.Equal(t, 1, messageDoneCount)
}

func TestCodexImageArtifactStreamStorageFailurePassesThrough(t *testing.T) {
	dataRoot := t.TempDir()
	blockingFile := filepath.Join(dataRoot, "not-a-directory")
	require.NoError(t, os.WriteFile(blockingFile, []byte("x"), 0o600))
	store := &codexGeneratedImageStore{
		dir:             filepath.Join(blockingFile, codexGeneratedImageDirectory),
		maxDecodedBytes: codexGeneratedImageMaxDecodedSize,
		ttl:             codexGeneratedImageTTL,
		cleanupEvery:    codexGeneratedImageCleanupEvery,
	}
	fallback := newCodexImageArtifactFallback(store, "https://api.example.test")
	imageItem := fmt.Sprintf(
		`{"id":"ig_failure","type":"image_generation_call","status":"completed","result":%s,"output_format":"png"}`,
		strconv.Quote(codexTestPNGBase64),
	)

	output := runCodexImageArtifactStreamTest(t, codexImageArtifactTestSSE(imageItem), fallback)
	payloads := collectCodexImageArtifactPayloads(output)

	for _, payload := range payloads {
		require.NotEqual(t, "response.output_text.delta", gjson.Get(payload, "type").String())
	}
	terminal := findCodexImageArtifactPayload(t, payloads, "response.completed", "")
	require.Equal(t, int64(7), gjson.Get(terminal, "sequence_number").Int())
	require.Empty(t, gjson.Get(terminal, "response.output").Array())
}

func TestCodexImageArtifactNonStreamingResponseAppendsMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.Pricing.DataDir = t.TempDir()
	svc := &OpenAIGatewayService{cfg: cfg}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "http://api.example.test/v1/responses", nil)
	c.Request.Host = "api.example.test"
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	setCodexImageArtifactFallbackActive(c, true)

	body := []byte(fmt.Sprintf(
		`{"id":"resp_nonstream","object":"response","status":"completed","model":"gpt-5.4","output":[{"id":"ig_nonstream","type":"image_generation_call","status":"completed","result":%s,"output_format":"png"}],"usage":{"input_tokens":1,"output_tokens":2}}`,
		strconv.Quote(codexTestPNGBase64),
	))
	updated := svc.applyCodexImageArtifactFallbackToResponse(c, body)

	require.Len(t, gjson.GetBytes(updated, "output").Array(), 2)
	require.Equal(t, "message", gjson.GetBytes(updated, "output.1.type").String())
	require.Contains(t, gjson.GetBytes(updated, "output.1.content.0.text").String(), "https://api.example.test/v1/generated-images/")
	files, err := os.ReadDir(filepath.Join(cfg.Pricing.DataDir, codexGeneratedImageDirectory))
	require.NoError(t, err)
	require.Len(t, files, 1)
}
