package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// AccountTestModeCustomText sends a user-provided paragraph instead of the
	// regular "hi" connectivity probe. GPT / OpenAI accounts only.
	AccountTestModeCustomText = "custom_text"
	// AccountTestModeDraw asks GPT for a playable SVG (HTML+SVG), not a raster image.
	AccountTestModeDraw = "draw"

	accountTestCollectDrawSVGKey = "account_test_collect_draw_svg"

	defaultOpenAICustomTextPrompt = "请用一段不超过80字的生动文字，描写一只鹈鹕第一次骑自行车的情景，结尾带一个意外的小转折。"
	// defaultOpenAIDrawPrompt is the user-provided drawing prompt.
	defaultOpenAIDrawPrompt = "创建一个 HTML，内容是 SVG 绘制一个鹈鹕骑自行车的 2D 动画。"
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

func resolveOpenAIDrawPrompt(prompt string) string {
	if text := strings.TrimSpace(prompt); text != "" {
		return text
	}
	return defaultOpenAIDrawPrompt
}

func markAccountTestCollectDrawSVG(c *gin.Context) {
	c.Set(accountTestCollectDrawSVGKey, true)
}

func accountTestCollectDrawSVG(c *gin.Context) bool {
	value, ok := c.Get(accountTestCollectDrawSVGKey)
	if !ok {
		return false
	}
	enabled, _ := value.(bool)
	return enabled
}

func extractFencedBlock(raw, lang string) string {
	marker := "```" + lang
	lower := strings.ToLower(raw)
	start := strings.Index(lower, marker)
	if start < 0 {
		return ""
	}
	from := start + len(marker)
	if nl := strings.IndexByte(raw[from:], '\n'); nl >= 0 {
		from += nl + 1
	} else if space := strings.IndexByte(raw[from:], ' '); space >= 0 {
		from += space + 1
	}
	endRel := strings.Index(raw[from:], "```")
	if endRel < 0 {
		return strings.TrimSpace(raw[from:])
	}
	return strings.TrimSpace(raw[from : from+endRel])
}

func stripMarkdownCodeFence(raw string) string {
	text := strings.TrimSpace(raw)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	text = strings.TrimPrefix(text, "```")
	if nl := strings.IndexByte(text, '\n'); nl >= 0 {
		text = text[nl+1:]
	}
	if idx := strings.LastIndex(text, "```"); idx >= 0 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}

func extractHTMLDocument(text string) string {
	lower := strings.ToLower(text)
	start := strings.Index(lower, "<!doctype")
	if start < 0 {
		start = strings.Index(lower, "<html")
	}
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(lower, "</html>")
	if end >= start {
		return strings.TrimSpace(text[start : end+len("</html>")])
	}
	return strings.TrimSpace(text[start:])
}

func extractSVGDocument(text string) string {
	lower := strings.ToLower(text)
	start := strings.Index(lower, "<svg")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(lower, "</svg>")
	if end < start {
		return ""
	}
	return strings.TrimSpace(text[start : end+len("</svg>")])
}

func wrapSVGInHTML(svg string) string {
	return `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><style>html,body{margin:0;height:100%;background:#fff;overflow:hidden}svg{display:block;width:100%;height:100%}</style></head><body>` + svg + `</body></html>`
}

func extractPlayableSVGHTML(raw string) (string, bool) {
	if html := extractFencedBlock(raw, "html"); html != "" {
		if doc := extractHTMLDocument(html); doc != "" {
			return doc, true
		}
		if svg := extractSVGDocument(html); svg != "" {
			return wrapSVGInHTML(svg), true
		}
	}
	if svg := extractFencedBlock(raw, "svg"); svg != "" {
		if extracted := extractSVGDocument(svg); extracted != "" {
			return wrapSVGInHTML(extracted), true
		}
		if strings.Contains(strings.ToLower(svg), "<svg") {
			return wrapSVGInHTML(svg), true
		}
	}

	text := stripMarkdownCodeFence(raw)
	if doc := extractHTMLDocument(text); doc != "" {
		return doc, true
	}
	if svg := extractSVGDocument(text); svg != "" {
		return wrapSVGInHTML(svg), true
	}
	return "", false
}

func (s *AccountTestService) maybeEmitCollectedDrawSVG(c *gin.Context, collected string) {
	if !accountTestCollectDrawSVG(c) {
		return
	}
	html, ok := extractPlayableSVGHTML(collected)
	if !ok {
		s.sendEvent(c, TestEvent{Type: "status", Text: "模型已返回文本，但没有解析出可播放的 SVG。"})
		return
	}
	s.sendEvent(c, TestEvent{Type: "svg", Text: html, MimeType: "text/html"})
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
