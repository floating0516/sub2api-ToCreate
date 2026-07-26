package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *OpenAIGatewayHandler) GeneratedImage(c *gin.Context) {
	if h == nil || h.gatewayService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	h.gatewayService.ServeCodexGeneratedImage(c)
}
