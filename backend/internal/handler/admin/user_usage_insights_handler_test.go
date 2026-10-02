//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userInsightsAdminStub struct {
	service.AdminService
	err   error
	calls int
}

func (s *userInsightsAdminStub) GetUserUsageInsights(context.Context, []int64) (map[int64]service.UserUsageInsights, error) {
	s.calls++
	return map[int64]service.UserUsageInsights{1: {Summary: &service.UserUsageSummary{TotalRequests: 9}}}, s.err
}

func TestUserUsageInsightsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, body    string
		fail          bool
		status, calls int
	}{
		{"valid", `{"user_ids":[1]}`, false, http.StatusOK, 1},
		{"empty", `{"user_ids":[]}`, false, http.StatusBadRequest, 0},
		{"negative", `{"user_ids":[-1]}`, false, http.StatusBadRequest, 0},
		{"oversized", `{"user_ids":[` + strings.Repeat("1,", 100) + `1]}`, false, http.StatusBadRequest, 0},
		{"failure", `{"user_ids":[1]}`, true, http.StatusServiceUnavailable, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &userInsightsAdminStub{}
			if tc.fail {
				svc.err = errors.New("unavailable")
			}
			h := NewUserHandler(svc, nil, nil, nil, nil, nil, nil)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/usage-insights", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			h.GetUsageInsights(c)
			require.Equal(t, tc.status, recorder.Code)
			require.Equal(t, tc.calls, svc.calls)
			if tc.status == http.StatusOK {
				var body struct {
					Data struct {
						Insights map[string]struct {
							Summary struct {
								Total int64 `json:"total_requests"`
							} `json:"usage_summary"`
						} `json:"insights"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				require.Equal(t, int64(9), body.Data.Insights["1"].Summary.Total)
				require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
			}
		})
	}
}
