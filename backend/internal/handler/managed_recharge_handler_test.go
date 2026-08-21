package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestManagedRechargeUserOrderOmitsProviderFields(t *testing.T) {
	order := managedRechargeUserOrder(&service.ManagedRechargeOrder{
		CDKMasked:      "abcd...wxyz",
		UpstreamStatus: "provider-internal-status",
		UserEmail:      "user@example.com",
		Username:       "user-name",
	})
	payload, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("marshal managed recharge user order: %v", err)
	}
	encoded := string(payload)
	for _, forbidden := range []string{"cdk_masked", "upstream_status", "user_email", "username", "provider-internal-status", "abcd...wxyz"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("managed recharge user order leaked %q: %s", forbidden, encoded)
		}
	}
}
