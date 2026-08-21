package service

import "testing"

func TestNormalizeManagedRechargePlanType(t *testing.T) {
	tests := map[string]string{
		"plus":         "plus",
		"ChatGPT_Plus": "plus",
		"pro":          "pro",
		"chatgpt-pro":  "pro",
		"team":         "",
		"":             "",
	}
	for input, expected := range tests {
		if actual := normalizeManagedRechargePlanType(input); actual != expected {
			t.Fatalf("normalizeManagedRechargePlanType(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestNormalizeManagedRechargeProductRequiresPlanType(t *testing.T) {
	input := ManagedRechargeProductInput{
		Slug:        "chatgpt_plus",
		Name:        "ChatGPT Plus",
		Description: "Managed recharge",
		Price:       10,
		Active:      true,
	}
	if _, err := normalizeManagedRechargeProduct(input); err == nil {
		t.Fatal("normalizeManagedRechargeProduct accepted a product without plan_type")
	}

	input.PlanType = "ChatGPT_Plus"
	normalized, err := normalizeManagedRechargeProduct(input)
	if err != nil {
		t.Fatalf("normalizeManagedRechargeProduct returned error: %v", err)
	}
	if normalized.PlanType != "plus" {
		t.Fatalf("normalized plan_type = %q, want plus", normalized.PlanType)
	}
}
