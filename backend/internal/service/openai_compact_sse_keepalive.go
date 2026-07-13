package service

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// openAICompactSSEKeepaliveKey stores the Responses compact downstream
	// keepalive state for the current request.
	openAICompactSSEKeepaliveKey = "openai_compact_sse_keepalive"
	// openAIMessagesSSEKeepaliveKey stores the Anthropic Messages downstream
	// keepalive state across OpenAI account retries.
	openAIMessagesSSEKeepaliveKey = "openai_messages_sse_keepalive"
)

var (
	openAICompactKeepalivePayload  = []byte(": keepalive\n\n")
	openAIMessagesKeepalivePayload = []byte("event: ping\ndata: {\"type\":\"ping\"}\n\n")
)

// openAISSEKeepaliveState survives individual upstream attempts so heartbeat
// bytes remain excluded from failover's "response already written" check.
type openAISSEKeepaliveState struct {
	mu        sync.Mutex
	writer    gin.ResponseWriter
	active    *openAISSEKeepalive
	committed bool
	bytes     int
}

// openAISSEKeepalive writes protocol-safe downstream heartbeats while an
// upstream request is waiting for response headers. A fresh instance is used
// for each upstream attempt; the request-scoped state above is shared.
type openAISSEKeepalive struct {
	state   *openAISSEKeepaliveState
	writer  gin.ResponseWriter
	payload []byte
	stopped bool
	stop    chan struct{}
}

// Keep the existing internal names available for compact tests and callers.
type openAICompactSSEKeepalive = openAISSEKeepalive
type openAICompactKeepaliveWriter = openAISSEKeepaliveWriter

// StartOpenAICompactSSEKeepalive starts SSE comment heartbeats while a
// body-signal compact request waits for the unary upstream response.
func StartOpenAICompactSSEKeepalive(c *gin.Context, interval time.Duration) func() {
	if !openAICompactClientWantsStream(c) {
		return func() {}
	}
	return startOpenAISSEKeepalive(c, openAICompactSSEKeepaliveKey, interval, openAICompactKeepalivePayload)
}

// StartOpenAIMessagesSSEKeepalive starts Anthropic ping events while an OpenAI
// /v1/messages bridge request waits for upstream response headers. The caller
// must invoke the returned stop function before starting another attempt.
func StartOpenAIMessagesSSEKeepalive(c *gin.Context, interval time.Duration) func() {
	return startOpenAISSEKeepalive(c, openAIMessagesSSEKeepaliveKey, interval, openAIMessagesKeepalivePayload)
}

func startOpenAISSEKeepalive(c *gin.Context, key string, interval time.Duration, payload []byte) func() {
	if c == nil || c.Writer == nil || interval <= 0 || len(payload) == 0 {
		return func() {}
	}

	state := getOrCreateOpenAISSEKeepaliveState(c, key)
	originalWriter := c.Writer
	alreadyWritten := originalWriter.Written()
	writtenSize := -1
	if alreadyWritten {
		writtenSize = originalWriter.Size()
	}
	k := &openAISSEKeepalive{
		state:   state,
		writer:  originalWriter,
		payload: payload,
		stop:    make(chan struct{}),
	}

	state.mu.Lock()
	if state.active != nil {
		state.active.markStoppedLocked()
	}
	state.writer = originalWriter
	state.active = k
	// A concurrency-wait keepalive may already have committed an SSE 200 before
	// the upstream attempt starts. Preserve that response state.
	if alreadyWritten && !state.committed {
		state.committed = true
		// Anything written before Forward starts is a scheduler/concurrency
		// keepalive, not model output. Include it in the transport-only byte
		// budget so failover and fallback checks remain accurate.
		if writtenSize > 0 {
			state.bytes += writtenSize
		}
	}
	state.mu.Unlock()

	wrappedWriter := &openAISSEKeepaliveWriter{ResponseWriter: originalWriter, k: k}
	c.Writer = wrappedWriter

	var reqDone <-chan struct{}
	if c.Request != nil {
		reqDone = c.Request.Context().Done()
	}
	go func() {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			select {
			case <-k.stop:
				return
			case <-reqDone:
				return
			case <-timer.C:
			}
			if !k.beat() {
				return
			}
			timer.Reset(interval)
		}
	}()

	return func() {
		k.Stop()
		// Do not leave a pooled middleware writer reachable through the wrapper
		// after this upstream attempt finishes.
		if current, ok := c.Writer.(*openAISSEKeepaliveWriter); ok && current == wrappedWriter {
			c.Writer = originalWriter
		}
	}
}

func getOrCreateOpenAISSEKeepaliveState(c *gin.Context, key string) *openAISSEKeepaliveState {
	if value, ok := c.Get(key); ok {
		if state, ok := value.(*openAISSEKeepaliveState); ok && state != nil {
			return state
		}
	}
	state := &openAISSEKeepaliveState{}
	c.Set(key, state)
	return state
}

// beat commits SSE headers on the first heartbeat and writes one protocol-safe
// heartbeat block. It returns false when this attempt is no longer active.
func (k *openAISSEKeepalive) beat() bool {
	if k == nil || k.state == nil {
		return false
	}
	state := k.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if k.stopped || state.active != k || k.writer == nil {
		return false
	}
	if !state.committed {
		header := k.writer.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-cache")
		header.Set("Connection", "keep-alive")
		header.Set("X-Accel-Buffering", "no")
		k.writer.WriteHeader(http.StatusOK)
		state.committed = true
	}
	n, err := k.writer.Write(k.payload)
	state.bytes += n
	if err != nil {
		k.markStoppedLocked()
		return false
	}
	k.writer.Flush()
	return true
}

// Stop stops this attempt's heartbeat. It is safe to call concurrently and is
// idempotent.
func (k *openAISSEKeepalive) Stop() {
	if k == nil || k.state == nil {
		return
	}
	k.state.mu.Lock()
	k.markStoppedLocked()
	k.state.mu.Unlock()
}

// markStoppedLocked requires k.state.mu to be held.
func (k *openAISSEKeepalive) markStoppedLocked() {
	if k == nil || k.stopped {
		return
	}
	k.stopped = true
	if k.state != nil && k.state.active == k {
		k.state.active = nil
	}
	close(k.stop)
}

// StopOpenAICompactSSEKeepaliveCommitted stops compact heartbeats and reports
// whether they committed HTTP 200.
func StopOpenAICompactSSEKeepaliveCommitted(c *gin.Context) bool {
	return stopOpenAISSEKeepaliveCommitted(c, openAICompactSSEKeepaliveKey)
}

// StopOpenAIMessagesSSEKeepaliveCommitted stops the active Messages heartbeat
// and reports whether any attempt committed HTTP 200.
func StopOpenAIMessagesSSEKeepaliveCommitted(c *gin.Context) bool {
	return stopOpenAISSEKeepaliveCommitted(c, openAIMessagesSSEKeepaliveKey)
}

func stopOpenAISSEKeepaliveCommitted(c *gin.Context, key string) bool {
	state := openAISSEKeepaliveStateFromContext(c, key)
	if state == nil {
		return false
	}
	state.mu.Lock()
	if state.active != nil {
		state.active.markStoppedLocked()
	}
	committed := state.committed
	state.mu.Unlock()
	return committed
}

// OpenAICompactKeepaliveAdjustedWrittenSize returns the response size with
// compact heartbeat bytes removed.
func OpenAICompactKeepaliveAdjustedWrittenSize(c *gin.Context) int {
	return openAISSEKeepaliveAdjustedWrittenSize(c, openAICompactSSEKeepaliveKey)
}

// OpenAIMessagesKeepaliveAdjustedWrittenSize returns the response size with
// early Anthropic ping bytes removed. This keeps account failover available
// while the client has only received heartbeats.
func OpenAIMessagesKeepaliveAdjustedWrittenSize(c *gin.Context) int {
	return openAISSEKeepaliveAdjustedWrittenSize(c, openAIMessagesSSEKeepaliveKey)
}

func openAISSEKeepaliveAdjustedWrittenSize(c *gin.Context, key string) int {
	if c == nil || c.Writer == nil {
		return -1
	}
	state := openAISSEKeepaliveStateFromContext(c, key)
	if state == nil {
		return c.Writer.Size()
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.writer == nil {
		return -1
	}
	size := state.writer.Size()
	if size < 0 {
		return size
	}
	if real := size - state.bytes; real > 0 {
		return real
	}
	return -1
}

func openAISSEKeepaliveStateFromContext(c *gin.Context, key string) *openAISSEKeepaliveState {
	if c == nil {
		return nil
	}
	value, ok := c.Get(key)
	if !ok {
		return nil
	}
	state, _ := value.(*openAISSEKeepaliveState)
	return state
}

// openAISSEKeepaliveWriter serializes request-side response construction with
// heartbeat writes. Read-only status methods do not suspend the heartbeat.
type openAISSEKeepaliveWriter struct {
	gin.ResponseWriter
	k *openAISSEKeepalive
}

func (w *openAISSEKeepaliveWriter) suspend() {
	if w.k != nil {
		w.k.Stop()
	}
}

func (w *openAISSEKeepaliveWriter) Header() http.Header {
	w.suspend()
	if w.ResponseWriter == nil {
		return http.Header{}
	}
	return w.ResponseWriter.Header()
}

func (w *openAISSEKeepaliveWriter) Write(data []byte) (int, error) {
	w.suspend()
	if w.ResponseWriter == nil {
		return 0, nil
	}
	return w.ResponseWriter.Write(data)
}

func (w *openAISSEKeepaliveWriter) WriteString(s string) (int, error) {
	w.suspend()
	if w.ResponseWriter == nil {
		return 0, nil
	}
	return w.ResponseWriter.WriteString(s)
}

func (w *openAISSEKeepaliveWriter) WriteHeader(code int) {
	w.suspend()
	if w.ResponseWriter != nil {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *openAISSEKeepaliveWriter) WriteHeaderNow() {
	w.suspend()
	if w.ResponseWriter != nil {
		w.ResponseWriter.WriteHeaderNow()
	}
}

func (w *openAISSEKeepaliveWriter) Flush() {
	w.suspend()
	if w.ResponseWriter != nil {
		w.ResponseWriter.Flush()
	}
}

func (w *openAISSEKeepaliveWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.ResponseWriter == nil {
		return nil, nil, errors.New("response writer released")
	}
	return w.ResponseWriter.Hijack()
}

func (w *openAISSEKeepaliveWriter) CloseNotify() <-chan bool {
	if w.ResponseWriter == nil {
		ch := make(chan bool)
		close(ch)
		return ch
	}
	return w.ResponseWriter.CloseNotify()
}

func (w *openAISSEKeepaliveWriter) Pusher() http.Pusher {
	if w.ResponseWriter == nil {
		return nil
	}
	return w.ResponseWriter.Pusher()
}

func (w *openAISSEKeepaliveWriter) Status() int {
	if w.k == nil || w.k.state == nil || w.ResponseWriter == nil {
		return 0
	}
	w.k.state.mu.Lock()
	defer w.k.state.mu.Unlock()
	return w.ResponseWriter.Status()
}

func (w *openAISSEKeepaliveWriter) Size() int {
	if w.k == nil || w.k.state == nil || w.ResponseWriter == nil {
		return 0
	}
	w.k.state.mu.Lock()
	defer w.k.state.mu.Unlock()
	return w.ResponseWriter.Size()
}

func (w *openAISSEKeepaliveWriter) Written() bool {
	if w.k == nil || w.k.state == nil || w.ResponseWriter == nil {
		return false
	}
	w.k.state.mu.Lock()
	defer w.k.state.mu.Unlock()
	return w.ResponseWriter.Written()
}
