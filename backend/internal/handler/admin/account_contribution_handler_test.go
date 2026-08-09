package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type contributionAdminServiceStub struct {
	service.AdminService
	overview *service.AccountContributionAdminOverview
	err      error
}

func (s *contributionAdminServiceStub) GetAccountContributionOverview(context.Context) (*service.AccountContributionAdminOverview, error) {
	return s.overview, s.err
}

func TestAccountHandlerGetContributionOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &AccountHandler{adminService: &contributionAdminServiceStub{
		overview: &service.AccountContributionAdminOverview{
			Features: service.AccountContributionFeatureState{Enabled: false},
			Stats:    service.AccountContributionAdminStats{ContributorsTotal: 2},
		},
	}}
	router := gin.New()
	router.GET("/api/v1/admin/account-contributions/overview", handler.GetContributionOverview)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/account-contributions/overview", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
		"code": 0,
		"message": "success",
		"data": {
			"features": {
				"enabled": false,
				"submission_configured": false,
				"payout_configured": false,
				"submission_enabled": false,
				"payout_enabled": false
			},
			"stats": {
				"contributors_total": 2,
				"contributors_pending": 0,
				"contributions_total": 0,
				"contributions_active": 0,
				"earning_entries_total": 0,
				"total_earnings_cny_fen": 0,
				"available_earnings_cny_fen": 0,
				"payout_requests_total": 0,
				"payout_requests_pending": 0,
				"pending_payout_cny_fen": 0
			},
			"contributors": null,
			"contributions": null,
			"earnings": null,
			"payouts": null
		}
	}`, recorder.Body.String())
}
