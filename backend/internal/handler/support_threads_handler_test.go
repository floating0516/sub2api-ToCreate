//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupportThreads_PaginationCannotOverrideOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	captured := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":2,"page_size":10,"pages":0}`))
	}))
	defer upstream.Close()
	h := newSupportHandlerForTest(upstream.URL, true)
	c, recorder := newSupportContext(http.MethodGet, "/api/v1/support/threads?page=2&page_size=10&user_id=456&role=agent", "")
	h.ListThreads(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "/threads?page=2&page_size=10", <-captured)
}

func TestSupportThreads_RejectsInvalidPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newSupportHandlerForTest("http://127.0.0.1:1", true)
	for _, query := range []string{"page=0", "page=-1", "page=abc", "page=100001", "page_size=51", "page_size=0"} {
		c, recorder := newSupportContext(http.MethodGet, "/api/v1/support/threads?"+query, "")
		h.ListThreads(c)
		require.Equal(t, http.StatusBadRequest, recorder.Code, query)
	}
}
