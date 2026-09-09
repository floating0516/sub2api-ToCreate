package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newOpenAIExtendedTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/admin/accounts/1/test", nil)
	return ctx, recorder
}

func TestNormalizeOpenAIAccountTestMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: AccountTestModeDefault},
		{input: "default", want: AccountTestModeDefault},
		{input: "compact", want: AccountTestModeCompact},
		{input: "custom_text", want: AccountTestModeCustomText},
		{input: " DRAW ", want: AccountTestModeDraw},
		{input: "unknown", want: AccountTestModeDefault},
	}
	for _, tt := range tests {
		if got := normalizeOpenAIAccountTestMode(tt.input); got != tt.want {
			t.Fatalf("normalizeOpenAIAccountTestMode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCreateOpenAITestPayloadWithOptions_KeepsDefaultHi(t *testing.T) {
	t.Parallel()

	base := createOpenAITestPayload("gpt-5.4", true)
	got := createOpenAITestPayloadWithOptions("gpt-5.4", true, "", "")
	require.Equal(t, base, got)
}

func TestCreateOpenAITestPayloadWithOptions_CustomTextAndThinking(t *testing.T) {
	t.Parallel()

	payload := createOpenAITestPayloadWithOptions("gpt-5.4", true, "一段测试文字", "high")
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"text":"一段测试文字"`)
	require.Contains(t, string(raw), `"effort":"high"`)
	require.NotContains(t, string(raw), `"text":"hi"`)
}

func TestResolveOpenAICustomTextPrompt(t *testing.T) {
	t.Parallel()
	require.Equal(t, defaultOpenAICustomTextPrompt, resolveOpenAICustomTextPrompt("  "))
	require.Equal(t, "hello", resolveOpenAICustomTextPrompt(" hello "))
}

func TestResolveOpenAIDrawPrompt(t *testing.T) {
	t.Parallel()
	require.Equal(t, defaultOpenAIDrawPrompt, resolveOpenAIDrawPrompt("  "))
	require.Equal(t, "创建一个 HTML，内容是 SVG 绘制一个鹈鹕骑自行车的 2D 动画。", defaultOpenAIDrawPrompt)
}

func TestExtractPlayableSVGHTML(t *testing.T) {
	t.Parallel()

	html, ok := extractPlayableSVGHTML("```html\n<html><body><svg><circle r='4'/></svg></body></html>\n```")
	require.True(t, ok)
	require.Contains(t, html, "<svg>")
	require.NotContains(t, html, "```")

	wrapped, ok := extractPlayableSVGHTML("here you go\n<svg viewBox='0 0 10 10'></svg>\nthanks")
	require.True(t, ok)
	require.Contains(t, wrapped, "<!DOCTYPE html>")
	require.Contains(t, wrapped, "<svg viewBox='0 0 10 10'></svg>")

	_, ok = extractPlayableSVGHTML("just a paragraph")
	require.False(t, ok)
}

func TestMaybeEmitCollectedDrawSVG(t *testing.T) {
	t.Parallel()

	ctx, recorder := newOpenAIExtendedTestContext()
	svc := &AccountTestService{}
	svc.maybeEmitCollectedDrawSVG(ctx, "<svg id='p'></svg>")
	require.NotContains(t, recorder.Body.String(), `"type":"svg"`)

	markAccountTestCollectDrawSVG(ctx)
	svc.maybeEmitCollectedDrawSVG(ctx, "```svg\n<svg id='p'></svg>\n```")
	require.Contains(t, recorder.Body.String(), `"type":"svg"`)
	require.Contains(t, recorder.Body.String(), `"mime_type":"text/html"`)
	require.Contains(t, recorder.Body.String(), "<svg id='p'></svg>")
	require.NotContains(t, recorder.Body.String(), "image/png")
}
