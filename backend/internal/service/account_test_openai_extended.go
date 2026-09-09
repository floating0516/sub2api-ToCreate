package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// AccountTestModeCustomText sends a user-provided paragraph instead of the
	// regular "hi" connectivity probe. GPT / OpenAI accounts only.
	AccountTestModeCustomText = "custom_text"
	// AccountTestModeDraw returns a playable SVG (HTML+SVG), not a raster image.
	// GPT / OpenAI accounts only. The first staging build uses a canned pelican
	// bicycle animation until the real drawing prompt is wired.
	AccountTestModeDraw = "draw"

	defaultOpenAICustomTextPrompt = "请用一段不超过80字的生动文字，描写一只鹈鹕第一次骑自行车的情景，结尾带一个意外的小转折。"
)

func normalizeOpenAIAccountTestMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AccountTestModeCompact:
		return AccountTestModeCompact
	case AccountTestModeCustomText:
		return AccountTestModeCustomText
	case AccountTestModeDraw:
		return AccountTestModeDraw
	default:
		return AccountTestModeDefault
	}
}

func normalizeOpenAITestThinkingEffort(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "minimal", "low", "medium", "high", "xhigh":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func resolveOpenAICustomTextPrompt(prompt string) string {
	if text := strings.TrimSpace(prompt); text != "" {
		return text
	}
	return defaultOpenAICustomTextPrompt
}

func createOpenAITestPayloadWithOptions(modelID string, isOAuth bool, prompt, thinkingEffort string) map[string]any {
	payload := createOpenAITestPayload(modelID, isOAuth)
	if text := strings.TrimSpace(prompt); text != "" {
		payload["input"] = []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": text,
					},
				},
			},
		}
	}
	if effort := normalizeOpenAITestThinkingEffort(thinkingEffort); effort != "" {
		payload["reasoning"] = map[string]any{"effort": effort}
	}
	return payload
}

func (s *AccountTestService) emitOpenAIDrawSVGPreview(c *gin.Context, modelID string) error {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})
	s.sendEvent(c, TestEvent{
		Type: "status",
		Text: "绘图测试当前返回可播放 SVG，而不是位图。正式提示词稍后接入。",
	})
	s.sendEvent(c, TestEvent{
		Type:     "svg",
		Text:     openAIDrawPelicanBicycleHTML,
		MimeType: "text/html",
	})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
