package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newMessagesKeepaliveTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, recorder
}

func TestOpenAIMessagesSSEKeepalive_CommitsAnthropicPing(t *testing.T) {
	c, recorder := newMessagesKeepaliveTestContext(t)
	stop := StartOpenAIMessagesSSEKeepalive(c, keepaliveTestInterval)
	defer stop()
	waitForKeepaliveBeats()

	require.True(t, StopOpenAIMessagesSSEKeepaliveCommitted(c))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Equal(t, "no", recorder.Header().Get("X-Accel-Buffering"))
	require.Contains(t, recorder.Body.String(), "event: ping\ndata: {\"type\":\"ping\"}\n\n")
}

func TestOpenAIMessagesSSEKeepalive_StopBeforeFirstBeatKeepsHTTPErrorPath(t *testing.T) {
	c, recorder := newMessagesKeepaliveTestContext(t)
	stop := StartOpenAIMessagesSSEKeepalive(c, time.Hour)
	writeAnthropicError(c, http.StatusBadGateway, "api_error", "fast failure")
	stop()

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, "error", gjson.Get(recorder.Body.String(), "type").String())
	require.Equal(t, "fast failure", gjson.Get(recorder.Body.String(), "error.message").String())
	require.NotContains(t, recorder.Body.String(), "event: ping")
}

func TestOpenAIMessagesSSEKeepalive_CommittedErrorUsesAnthropicSSE(t *testing.T) {
	c, recorder := newMessagesKeepaliveTestContext(t)
	stop := StartOpenAIMessagesSSEKeepalive(c, keepaliveTestInterval)
	defer stop()
	waitForKeepaliveBeats()

	writeAnthropicError(c, http.StatusBadGateway, "api_error", "upstream failed")

	body := recorder.Body.String()
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, body, "event: ping\n")
	require.Contains(t, body, "event: error\n")
	require.Contains(t, body, `"message":"upstream failed"`)
	require.True(t, IsResponseCommitted(c))
	streamErr, ok := GetOpsStreamError(c)
	require.True(t, ok)
	require.Equal(t, http.StatusBadGateway, streamErr.IntendedStatus)
}

func TestOpenAIMessagesKeepaliveAdjustedWrittenSize_PersistsAcrossRetries(t *testing.T) {
	c, recorder := newMessagesKeepaliveTestContext(t)
	require.Equal(t, -1, OpenAIMessagesKeepaliveAdjustedWrittenSize(c))

	stopFirst := StartOpenAIMessagesSSEKeepalive(c, keepaliveTestInterval)
	waitForKeepaliveBeats()
	stopFirst()
	require.Equal(t, -1, OpenAIMessagesKeepaliveAdjustedWrittenSize(c))

	stopSecond := StartOpenAIMessagesSSEKeepalive(c, keepaliveTestInterval)
	waitForKeepaliveBeats()
	stopSecond()
	require.GreaterOrEqual(t, strings.Count(recorder.Body.String(), "event: ping\n"), 2)
	require.Equal(t, -1, OpenAIMessagesKeepaliveAdjustedWrittenSize(c))

	_, err := c.Writer.Write([]byte("real-response"))
	require.NoError(t, err)
	require.Equal(t, len("real-response"), OpenAIMessagesKeepaliveAdjustedWrittenSize(c))
}

func TestOpenAIMessagesKeepaliveAdjustedWrittenSize_ExcludesPriorWaitPing(t *testing.T) {
	c, recorder := newMessagesKeepaliveTestContext(t)
	c.Header("Content-Type", "text/event-stream")
	c.Status(http.StatusOK)
	_, err := c.Writer.WriteString("event: ping\ndata: {\"type\":\"ping\"}\n\n")
	require.NoError(t, err)
	c.Writer.Flush()

	stop := StartOpenAIMessagesSSEKeepalive(c, time.Hour)
	defer stop()

	require.True(t, StopOpenAIMessagesSSEKeepaliveCommitted(c))
	require.Equal(t, -1, OpenAIMessagesKeepaliveAdjustedWrittenSize(c))
	require.Contains(t, recorder.Body.String(), "event: ping\n")
}
