package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestInstallScriptsUseRequestOriginWithoutEmbeddingSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerInstallScriptRoutes(router)

	tests := []struct {
		path        string
		contentType string
	}{
		{"/install.sh", "text/x-shellscript"},
		{"/install.ps1", "text/plain"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Host = "api.example.com"
			request.Header.Set("X-Forwarded-Proto", "https")
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			require.Contains(t, recorder.Header().Get("Content-Type"), test.contentType)
			require.Contains(t, recorder.Body.String(), "https://api.example.com")
			require.Contains(t, recorder.Body.String(), "/api/v1/install-token/redeem")
			require.NotContains(t, recorder.Body.String(), installScriptBaseURLPlaceholder)
			require.NotContains(t, recorder.Body.String(), "sk-test")
		})
	}
}
