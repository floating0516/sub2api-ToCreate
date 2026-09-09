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

func TestEmitOpenAIDrawSVGPreview(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	ctx, recorder := newOpenAIExtendedTestContext()
	svc := &AccountTestService{}
	require.NoError(t, svc.emitOpenAIDrawSVGPreview(ctx, "gpt-5.4"))

	body := recorder.Body.String()
	require.Contains(t, body, `"type":"svg"`)
	require.Contains(t, body, `"mime_type":"text/html"`)
	require.Contains(t, body, "鹈鹕骑自行车")
	require.Contains(t, body, "<svg")
	require.NotContains(t, body, "image/png")
	require.True(t, strings.Contains(body, `"type":"test_complete"`))
}
