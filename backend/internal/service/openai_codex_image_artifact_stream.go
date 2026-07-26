package service

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const codexImageArtifactFallbackContextKey = "codex_image_artifact_fallback_active"

type codexGeneratedImageLink struct {
	artifact codexGeneratedImageArtifact
	url      string
}

type codexCapturedImageOutput struct {
	key string
	raw json.RawMessage
}

type codexImageArtifactFallback struct {
	store  *codexGeneratedImageStore
	origin string

	attempted    map[string]struct{}
	imageOutputs []codexCapturedImageOutput
	links        []codexGeneratedImageLink
	hasSequence  bool
	maxSequence  int64
	injected     bool
}

func setCodexImageArtifactFallbackActive(c *gin.Context, active bool) {
	if c == nil {
		return
	}
	c.Set(codexImageArtifactFallbackContextKey, active)
}

func isCodexImageArtifactFallbackActive(c *gin.Context) bool {
	if c == nil {
		return false
	}
	value, ok := c.Get(codexImageArtifactFallbackContextKey)
	active, valid := value.(bool)
	return ok && valid && active
}

func newCodexImageArtifactFallback(store *codexGeneratedImageStore, origin string) *codexImageArtifactFallback {
	if store == nil || strings.TrimSpace(origin) == "" {
		return nil
	}
	return &codexImageArtifactFallback{
		store:     store,
		origin:    strings.TrimRight(origin, "/"),
		attempted: make(map[string]struct{}),
	}
}

func codexGeneratedImagePublicOrigin(c *gin.Context) (string, error) {
	if c == nil || c.Request == nil {
		return "", fmt.Errorf("generated image public URL: request is unavailable")
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" || strings.ContainsAny(host, "@/?#\\\r\n\t ") {
		return "", fmt.Errorf("generated image public URL: request host is invalid")
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		candidate := strings.ToLower(strings.TrimSpace(strings.Split(forwarded, ",")[0]))
		if candidate == "http" || candidate == "https" {
			scheme = candidate
		}
	}
	parsed, err := url.Parse(scheme + "://" + host)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" {
		return "", fmt.Errorf("generated image public URL: request origin is invalid")
	}
	return parsed.String(), nil
}

func (f *codexImageArtifactFallback) capturePayload(payload []byte) {
	if f == nil || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return
	}
	switch strings.TrimSpace(gjson.GetBytes(payload, "type").String()) {
	case "response.output_item.done":
		f.captureItem(gjson.GetBytes(payload, "item"))
	case "response.completed", "response.done":
		for _, item := range gjson.GetBytes(payload, "response.output").Array() {
			f.captureItem(item)
		}
	}
}

func (f *codexImageArtifactFallback) captureResponse(payload []byte) {
	if f == nil || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return
	}
	for _, item := range gjson.GetBytes(payload, "output").Array() {
		f.captureItem(item)
	}
}

func (f *codexImageArtifactFallback) captureItem(item gjson.Result) {
	if f == nil || !item.Exists() || !item.IsObject() ||
		strings.TrimSpace(item.Get("type").String()) != "image_generation_call" {
		return
	}
	result := strings.TrimSpace(item.Get("result").String())
	if result == "" {
		return
	}
	key := codexImageGenerationOutputKey(item, result)
	if key == "" {
		return
	}
	if _, exists := f.attempted[key]; exists {
		return
	}
	f.attempted[key] = struct{}{}
	f.imageOutputs = append(f.imageOutputs, codexCapturedImageOutput{
		key: key,
		raw: json.RawMessage(item.Raw),
	})

	artifact, err := f.store.SaveBase64(result, firstNonEmpty(
		item.Get("output_format").String(),
		item.Get("format").String(),
	))
	if err != nil {
		logger.LegacyPrintf("service.openai_gateway", "Codex generated image artifact save skipped: %v", err)
		return
	}
	f.links = append(f.links, codexGeneratedImageLink{
		artifact: artifact,
		url:      f.origin + codexGeneratedImageRoutePrefix + url.PathEscape(artifact.Name),
	})
}

func codexImageGenerationOutputKey(item gjson.Result, result string) string {
	if id := strings.TrimSpace(item.Get("id").String()); id != "" {
		return "id:" + id
	}
	sum := sha256.Sum256([]byte(result))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (f *codexImageArtifactFallback) observeSequence(payload []byte) {
	if f == nil || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return
	}
	sequence := gjson.GetBytes(payload, "sequence_number")
	if !sequence.Exists() || sequence.Int() < 0 {
		return
	}
	value := sequence.Int()
	if !f.hasSequence || value > f.maxSequence {
		f.maxSequence = value
	}
	f.hasSequence = true
}

func (f *codexImageArtifactFallback) transformEvent(payload []byte) [][]byte {
	if f == nil || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return [][]byte{payload}
	}
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	switch eventType {
	case "response.completed", "response.done":
		f.capturePayload(payload)
		if f.injected || len(f.links) == 0 {
			f.observeSequence(payload)
			return [][]byte{payload}
		}
		events, terminal, err := f.buildTerminalFallback(payload)
		if err != nil {
			logger.LegacyPrintf("service.openai_gateway", "Codex generated image fallback injection skipped: %v", err)
			f.observeSequence(payload)
			return [][]byte{payload}
		}
		f.injected = true
		return append(events, terminal)
	default:
		f.capturePayload(payload)
		f.observeSequence(payload)
		return [][]byte{payload}
	}
}

func (f *codexImageArtifactFallback) buildTerminalFallback(payload []byte) ([][]byte, []byte, error) {
	messageText := f.markdownMessage()
	messageID := f.messageID()
	messageItem := map[string]any{
		"id":     messageID,
		"type":   "message",
		"status": "completed",
		"role":   "assistant",
		"content": []any{
			map[string]any{
				"type":        "output_text",
				"text":        messageText,
				"annotations": []any{},
			},
		},
	}
	messageJSON, err := json.Marshal(messageItem)
	if err != nil {
		return nil, nil, fmt.Errorf("encode generated image message: %w", err)
	}

	output, outputIndex, err := f.terminalOutputWithMessage(payload, messageJSON)
	if err != nil {
		return nil, nil, err
	}
	terminal, err := sjson.SetRawBytes(payload, "response.output", output)
	if err != nil {
		return nil, nil, fmt.Errorf("append generated image message: %w", err)
	}

	terminalSequence := gjson.GetBytes(payload, "sequence_number")
	numbered := f.hasSequence || terminalSequence.Exists()
	nextSequence := int64(0)
	if f.hasSequence {
		nextSequence = f.maxSequence + 1
	}
	if terminalSequence.Exists() && terminalSequence.Int() > nextSequence {
		nextSequence = terminalSequence.Int()
	}

	inProgressItem := map[string]any{
		"id":      messageID,
		"type":    "message",
		"status":  "in_progress",
		"role":    "assistant",
		"content": []any{},
	}
	emptyPart := map[string]any{
		"type":        "output_text",
		"text":        "",
		"annotations": []any{},
	}
	donePart := map[string]any{
		"type":        "output_text",
		"text":        messageText,
		"annotations": []any{},
	}
	eventTemplates := []map[string]any{
		{
			"type":         "response.output_item.added",
			"output_index": outputIndex,
			"item":         inProgressItem,
		},
		{
			"type":          "response.content_part.added",
			"item_id":       messageID,
			"output_index":  outputIndex,
			"content_index": 0,
			"part":          emptyPart,
		},
		{
			"type":          "response.output_text.delta",
			"item_id":       messageID,
			"output_index":  outputIndex,
			"content_index": 0,
			"delta":         messageText,
		},
		{
			"type":          "response.output_text.done",
			"item_id":       messageID,
			"output_index":  outputIndex,
			"content_index": 0,
			"text":          messageText,
		},
		{
			"type":          "response.content_part.done",
			"item_id":       messageID,
			"output_index":  outputIndex,
			"content_index": 0,
			"part":          donePart,
		},
		{
			"type":         "response.output_item.done",
			"output_index": outputIndex,
			"item":         messageItem,
		},
	}

	events := make([][]byte, 0, len(eventTemplates))
	for _, event := range eventTemplates {
		if numbered {
			event["sequence_number"] = nextSequence
			nextSequence++
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return nil, nil, fmt.Errorf("encode generated image fallback event: %w", err)
		}
		events = append(events, encoded)
	}
	if numbered {
		terminal, err = sjson.SetBytes(terminal, "sequence_number", nextSequence)
		if err != nil {
			return nil, nil, fmt.Errorf("update generated image terminal sequence: %w", err)
		}
		f.hasSequence = true
		f.maxSequence = nextSequence
	}
	return events, terminal, nil
}

func (f *codexImageArtifactFallback) terminalOutputWithMessage(payload, messageJSON []byte) ([]byte, int, error) {
	output := make([]json.RawMessage, 0)
	existingKeys := make(map[string]struct{})
	for _, item := range gjson.GetBytes(payload, "response.output").Array() {
		raw := json.RawMessage(item.Raw)
		output = append(output, raw)
		if strings.TrimSpace(item.Get("type").String()) == "image_generation_call" {
			result := strings.TrimSpace(item.Get("result").String())
			if result != "" {
				existingKeys[codexImageGenerationOutputKey(item, result)] = struct{}{}
			}
		}
	}
	for _, captured := range f.imageOutputs {
		if _, exists := existingKeys[captured.key]; exists {
			continue
		}
		output = append(output, captured.raw)
		existingKeys[captured.key] = struct{}{}
	}
	outputIndex := len(output)
	output = append(output, json.RawMessage(messageJSON))
	encoded, err := json.Marshal(output)
	if err != nil {
		return nil, 0, fmt.Errorf("encode generated image terminal output: %w", err)
	}
	return encoded, outputIndex, nil
}

func (f *codexImageArtifactFallback) markdownMessage() string {
	var builder strings.Builder
	for index, link := range f.links {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		label := "Generated image"
		if len(f.links) > 1 {
			label = fmt.Sprintf("Generated image %d", index+1)
		}
		builder.WriteString(label)
		builder.WriteString(":\n\n![")
		builder.WriteString(label)
		builder.WriteString("](")
		builder.WriteString(link.url)
		builder.WriteString(")\n\nDownload: ")
		builder.WriteString(link.url)
	}
	return builder.String()
}

func (f *codexImageArtifactFallback) messageID() string {
	if len(f.links) == 0 {
		return "msg_codex_generated_image"
	}
	stem := strings.TrimSuffix(f.links[0].artifact.Name, filepath.Ext(f.links[0].artifact.Name))
	if len(stem) > 24 {
		stem = stem[:24]
	}
	return "msg_codex_image_" + stem
}

type codexImageArtifactStreamBody struct {
	*io.PipeReader
	source io.Closer
}

func (b *codexImageArtifactStreamBody) Close() error {
	readerErr := b.PipeReader.Close()
	sourceErr := b.source.Close()
	if readerErr != nil {
		return readerErr
	}
	return sourceErr
}

func (s *OpenAIGatewayService) wrapCodexImageArtifactStream(c *gin.Context, source io.ReadCloser) io.ReadCloser {
	if source == nil {
		return source
	}
	store := s.getCodexGeneratedImageStore()
	origin, err := codexGeneratedImagePublicOrigin(c)
	if store == nil || err != nil {
		if err != nil {
			logger.LegacyPrintf("service.openai_gateway", "Codex generated image public URL unavailable: %v", err)
		}
		return source
	}
	fallback := newCodexImageArtifactFallback(store, origin)
	reader, writer := io.Pipe()
	body := &codexImageArtifactStreamBody{PipeReader: reader, source: source}
	maxLineSize := defaultMaxLineSize
	if s != nil && s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	go transformCodexImageArtifactStream(source, writer, fallback, maxLineSize)
	return body
}

func transformCodexImageArtifactStream(
	source io.ReadCloser,
	destination *io.PipeWriter,
	fallback *codexImageArtifactFallback,
	maxLineSize int,
) {
	defer func() { _ = source.Close() }()
	if maxLineSize <= 0 {
		maxLineSize = defaultMaxLineSize
	}

	scanner := bufio.NewScanner(source)
	scanBuf := getSSEScannerBuf64K()
	defer putSSEScannerBuf64K(scanBuf)
	scanner.Buffer(scanBuf[:0], maxLineSize)
	documents := newOpenAISSEJSONDocumentScanner(scanner)
	buffered := bufio.NewWriterSize(destination, 4*1024)
	pendingFields := make([]string, 0, 2)
	frameHadEventField := false
	frameEmitted := false

	writeLine := func(line string) error {
		if _, err := buffered.WriteString(line); err != nil {
			return err
		}
		return buffered.WriteByte('\n')
	}
	writePendingFields := func(payload []byte, includeNonEvent bool) error {
		eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
		for _, field := range pendingFields {
			if _, isEvent := extractOpenAISSEEventLine(field); isEvent {
				if eventType != "" {
					if err := writeLine("event: " + eventType); err != nil {
						return err
					}
				} else if err := writeLine(field); err != nil {
					return err
				}
				continue
			}
			if includeNonEvent {
				if err := writeLine(field); err != nil {
					return err
				}
			}
		}
		return nil
	}
	writePayloads := func(payloads [][]byte) error {
		for index, payload := range payloads {
			if index == 0 {
				if err := writePendingFields(payload, true); err != nil {
					return err
				}
			} else if frameHadEventField {
				eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
				if eventType != "" {
					if err := writeLine("event: " + eventType); err != nil {
						return err
					}
				}
			}
			if err := writeLine("data: " + string(payload)); err != nil {
				return err
			}
			if err := writeLine(""); err != nil {
				return err
			}
		}
		return buffered.Flush()
	}

	for documents.Scan() {
		line := documents.Text()
		data, isData := extractOpenAISSEDataLine(line)
		if isData {
			payload := []byte(data)
			payloads := [][]byte{payload}
			if json.Valid(payload) {
				payloads = fallback.transformEvent(payload)
			}
			if err := writePayloads(payloads); err != nil {
				_ = destination.CloseWithError(err)
				return
			}
			pendingFields = pendingFields[:0]
			frameHadEventField = false
			frameEmitted = true
			continue
		}

		if line == "" {
			if !frameEmitted {
				for _, field := range pendingFields {
					if err := writeLine(field); err != nil {
						_ = destination.CloseWithError(err)
						return
					}
				}
				if len(pendingFields) > 0 {
					if err := writeLine(""); err != nil {
						_ = destination.CloseWithError(err)
						return
					}
					if err := buffered.Flush(); err != nil {
						_ = destination.CloseWithError(err)
						return
					}
				}
			}
			pendingFields = pendingFields[:0]
			frameHadEventField = false
			frameEmitted = false
			continue
		}

		if _, isEvent := extractOpenAISSEEventLine(line); isEvent {
			frameHadEventField = true
		}
		pendingFields = append(pendingFields, line)
	}

	for _, field := range pendingFields {
		if err := writeLine(field); err != nil {
			_ = destination.CloseWithError(err)
			return
		}
	}
	if err := buffered.Flush(); err != nil {
		_ = destination.CloseWithError(err)
		return
	}
	if err := documents.Err(); err != nil {
		_ = destination.CloseWithError(err)
		return
	}
	_ = destination.Close()
}

func (s *OpenAIGatewayService) applyCodexImageArtifactFallbackToResponse(c *gin.Context, body []byte) []byte {
	if !isCodexImageArtifactFallbackActive(c) || len(body) == 0 || !gjson.ValidBytes(body) {
		return body
	}
	store := s.getCodexGeneratedImageStore()
	origin, err := codexGeneratedImagePublicOrigin(c)
	if store == nil || err != nil {
		return body
	}
	fallback := newCodexImageArtifactFallback(store, origin)
	fallback.captureResponse(body)
	if len(fallback.links) == 0 {
		return body
	}
	messageJSON, err := json.Marshal(map[string]any{
		"id":     fallback.messageID(),
		"type":   "message",
		"status": "completed",
		"role":   "assistant",
		"content": []any{
			map[string]any{
				"type":        "output_text",
				"text":        fallback.markdownMessage(),
				"annotations": []any{},
			},
		},
	})
	if err != nil {
		return body
	}
	output := make([]json.RawMessage, 0)
	for _, item := range gjson.GetBytes(body, "output").Array() {
		output = append(output, json.RawMessage(item.Raw))
	}
	output = append(output, json.RawMessage(messageJSON))
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return body
	}
	updated, err := sjson.SetRawBytes(body, "output", outputJSON)
	if err != nil {
		return body
	}
	return updated
}
