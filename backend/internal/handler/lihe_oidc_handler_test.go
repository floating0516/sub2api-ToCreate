package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOIDCBrowserBindingUsesHostOnlySecureCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "https://api.lihe.chat/api/v1/oidc/prepare", nil)

	binding, err := ensureOIDCBrowserBinding(context)
	require.NoError(t, err)
	require.Len(t, binding, 43)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	require.Equal(t, "__Host-lihe_oidc_browser", cookie.Name)
	require.Equal(t, binding, cookie.Value)
	require.Equal(t, "/", cookie.Path)
	require.Empty(t, cookie.Domain)
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestOIDCBrowserBindingRejectsMalformedCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "https://api.lihe.chat/api/v1/oidc/authorize", nil)
	context.Request.AddCookie(&http.Cookie{Name: liheOIDCBrowserCookie, Value: "too-short"})

	_, err := readOIDCBrowserBinding(context)
	require.Error(t, err)
}
