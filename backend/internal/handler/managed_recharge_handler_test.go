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
		CDKMasked:      "abcd...wxyz",
		RedemptionCode: "USER-OWNED-CDK",
		RedemptionURL:  "https://redeem.example.test/recharge?cdk=USER-OWNED-CDK",
		Status:         service.ManagedRechargeStatusProcessing,
		UpstreamStatus: "provider-internal-status",
		Progress:       `{"step":"provider-checkout"}`,
		ErrorCode:      "UPSTREAM_INTERNAL_CODE",
		UserEmail:      "user@example.com",
		Username:       "user-name",
	})
	payload, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("marshal managed recharge user order: %v", err)
	}
	encoded := string(payload)
	for _, forbidden := range []string{"cdk_masked", "upstream_status", "error_code", "user_email", "username", "provider-internal-status", "provider-checkout", "UPSTREAM_INTERNAL_CODE", "abcd...wxyz"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("managed recharge user order leaked %q: %s", forbidden, encoded)
		}
	}
	for _, expected := range []string{"redemption_code", "USER-OWNED-CDK", "redemption_url", "https://redeem.example.test/recharge", "progress", "兑换服务正在处理"} {
		if !strings.Contains(encoded, expected) {
			t.Fatalf("managed recharge user order omitted %q: %s", expected, encoded)
		}
	}
}

func TestManagedRechargeUserProgressMapsProviderStepsWithoutLeakingRawPayload(t *testing.T) {
	tests := []struct {
		step     string
		expected string
	}{
		{step: "login_otp", expected: "正在验证目标账号"},
		{step: "checkout", expected: "正在创建订阅订单"},
		{step: "submitting", expected: "正在提交付款"},
		{step: "verifying", expected: "付款已完成，正在确认订阅状态"},
	}
	for _, test := range tests {
		order := &service.ManagedRechargeOrder{
			Status:   service.ManagedRechargeStatusProcessing,
			Progress: `{"step":"` + test.step + `","provider_detail":"secret"}`,
		}
		if actual := managedRechargeUserProgress(order); actual != test.expected {
			t.Fatalf("progress step %q = %q, want %q", test.step, actual, test.expected)
		}
	}
}

func TestManagedRechargeUserOrderStatusNeverReturnsCDK(t *testing.T) {
	order := managedRechargeUserOrderStatus(&service.ManagedRechargeOrder{
		RedemptionCode: "USER-OWNED-CDK",
		RedemptionURL:  "https://redeem.example.test/recharge?cdk=USER-OWNED-CDK",
		Status:         service.ManagedRechargeStatusIssued,
	})
	payload, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("marshal managed recharge status order: %v", err)
	}
	encoded := string(payload)
	for _, forbidden := range []string{"redemption_code", "redemption_url", "USER-OWNED-CDK", "redeem.example.test"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("managed recharge status response leaked %q: %s", forbidden, encoded)
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
