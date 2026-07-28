package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type InstallTokenHandler struct {
	service *service.InstallTokenService
}

type issueInstallTokenRequest struct {
	Client        string `json:"client"`
	KeyID         int64  `json:"key_id"`
	PreviousToken string `json:"previous_token"`
}

type installCredentialRequest struct {
	Token   string `json:"token"`
	Receipt string `json:"receipt"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func NewInstallTokenHandler(installTokenService *service.InstallTokenService) *InstallTokenHandler {
	return &InstallTokenHandler{service: installTokenService}
}

func (h *InstallTokenHandler) Issue(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var request issueInstallTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid install token request")
		return
	}
	result, err := h.service.Issue(
		c.Request.Context(),
		subject.UserID,
		request.Client,
		request.KeyID,
		request.PreviousToken,
		installRequestOrigin(c),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *InstallTokenHandler) Peek(c *gin.Context) {
	var request installCredentialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid install token request")
		return
	}
	credential := strings.TrimSpace(request.Token)
	if credential == "" {
		credential = strings.TrimSpace(request.Receipt)
	}
	result, err := h.service.Peek(
		c.Request.Context(),
		credential,
		middleware2.SecurityClientIP(c),
		installRequestOrigin(c),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *InstallTokenHandler) Redeem(c *gin.Context) {
	var request installCredentialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid install token request")
		return
	}
	if len(strings.TrimSpace(request.OS)) > 32 || len(strings.TrimSpace(request.Arch)) > 32 {
		response.BadRequest(c, "Invalid platform metadata")
		return
	}
	result, err := h.service.Redeem(
		c.Request.Context(),
		request.Token,
		middleware2.SecurityClientIP(c),
		installRequestOrigin(c),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *InstallTokenHandler) Confirm(c *gin.Context) {
	var request installCredentialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid install confirmation request")
		return
	}
	result, err := h.service.Confirm(
		c.Request.Context(),
		request.Receipt,
		middleware2.SecurityClientIP(c),
		installRequestOrigin(c),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *InstallTokenHandler) Revoke(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var request installCredentialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid install token request")
		return
	}
	status, err := h.service.Revoke(c.Request.Context(), subject.UserID, request.Token)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": status})
}

func installRequestOrigin(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	if strings.TrimSpace(c.Request.Host) == "" {
		return ""
	}
	return scheme + "://" + c.Request.Host
}
