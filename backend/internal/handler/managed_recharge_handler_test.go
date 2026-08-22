package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestManagedRechargeUserOrderOmitsProviderFields(t *testing.T) {
	order := managedRechargeUserOrder(&service.ManagedRechargeOrder{
		CDKMasked:       "abcd...wxyz",
		RedemptionCode:  "USER-OWNED-CDK",
		RedemptionURL:   "https://redeem.example.test/recharge?cdk=USER-OWNED-CDK",
		UpstreamStatus:  "provider-internal-status",
		Progress:        `{"step":"provider-checkout"}`,
		ErrorCode:       "UPSTREAM_INTERNAL_CODE",
		UserEmail:       "user@example.com",
		Username:        "user-name",
	})
	payload, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("marshal managed recharge user order: %v", err)
	}
	encoded := string(payload)
	for _, forbidden := range []string{"cdk_masked", "upstream_status", "progress", "error_code", "user_email", "username", "provider-internal-status", "provider-checkout", "UPSTREAM_INTERNAL_CODE", "abcd...wxyz"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("managed recharge user order leaked %q: %s", forbidden, encoded)
		}
	}
	for _, expected := range []string{"redemption_code", "USER-OWNED-CDK", "redemption_url", "https://redeem.example.test/recharge"} {
		if !strings.Contains(encoded, expected) {
			t.Fatalf("managed recharge user order omitted %q: %s", expected, encoded)
		}
	}
}

func TestManagedRechargeBindJSONRejectsOversizedSessionBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		var req managedRechargeCreateOrderRequest
		if managedRechargeBindJSON(c, &req, 64, "invalid") {
			c.Status(http.StatusNoContent)
		}
	})
	body := []byte(`{"product_id":1,"session_json":"` + strings.Repeat("x", 128) + `"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized managed recharge body status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}
