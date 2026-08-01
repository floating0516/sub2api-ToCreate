//go:build unit

package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestReadinessRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ready when all checks pass", func(t *testing.T) {
		r := gin.New()
		RegisterCommonRoutes(r, func(context.Context) error { return nil })

		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))

		require.Equal(t, http.StatusOK, response.Code)
		require.JSONEq(t, `{"status":"ready"}`, response.Body.String())
	})

	t.Run("not ready when a dependency fails", func(t *testing.T) {
		r := gin.New()
		RegisterCommonRoutes(r, func(context.Context) error { return errors.New("unavailable") })

		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))

		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		require.JSONEq(t, `{"status":"not_ready"}`, response.Body.String())
	})
}
