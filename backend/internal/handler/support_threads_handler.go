package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// ListThreads forwards only pagination. The Agent derives ownership from the
// server-issued token, never from browser query parameters.
func (h *SupportHandler) ListThreads(c *gin.Context) {
	if !h.supportReady(c, "/threads") {
		return
	}
	query := url.Values{}
	for _, field := range []struct {
		name string
		max  int
	}{
		{"page", 100000},
		{"page_size", 50},
	} {
		if value, exists := c.GetQuery(field.name); exists {
			number, err := strconv.Atoi(value)
			if err != nil || number < 1 || number > field.max {
				response.BadRequest(c, "invalid thread pagination")
				return
			}
			query.Set(field.name, strconv.Itoa(number))
		}
	}
	path := "/threads"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	h.proxyJSON(c, http.MethodGet, path, false)
}
