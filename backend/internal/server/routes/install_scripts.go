package routes

import (
	_ "embed"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const installScriptBaseURLPlaceholder = "__TOCREATE_BASE_URL__"

//go:embed install_scripts/install.sh
var unixInstallScript string

//go:embed install_scripts/install.ps1
var windowsInstallScript string

//go:embed install_scripts/install-config.js
var installConfigHelper string

func registerInstallScriptRoutes(r *gin.Engine) {
	r.GET("/install.sh", func(c *gin.Context) {
		serveInstallScript(c, unixInstallScript, "text/x-shellscript; charset=utf-8")
	})
	r.GET("/install.ps1", func(c *gin.Context) {
		serveInstallScript(c, windowsInstallScript, "text/plain; charset=utf-8")
	})
	r.GET("/install-config.js", func(c *gin.Context) {
		serveInstallScript(c, installConfigHelper, "application/javascript; charset=utf-8")
	})
}

func serveInstallScript(c *gin.Context, script string, contentType string) {
	baseURL := installScriptRequestOrigin(c)
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=300")
	c.String(http.StatusOK, strings.ReplaceAll(script, installScriptBaseURLPlaceholder, baseURL))
}

func installScriptRequestOrigin(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	candidate := scheme + "://" + strings.TrimSpace(c.Request.Host)
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Host == "" {
		return "https://api.lihe.chat"
	}
	return parsed.Scheme + "://" + parsed.Host
}
