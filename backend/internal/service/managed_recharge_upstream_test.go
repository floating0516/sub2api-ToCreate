package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagedRechargeUpstreamVerifyCDKUsesExpectedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/recharge/verify-cdk" {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode upstream payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload["cdk_code"] != "test-cdk" {
			t.Errorf("cdk_code = %q, want test-cdk", payload["cdk_code"])
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"plan_type":"plus"}`))
	}))
	defer server.Close()

	client := &managedRechargeUpstreamClient{baseURL: server.URL, client: server.Client()}
	result, err := client.verifyCDK(context.Background(), "test-cdk")
	if err != nil {
		t.Fatalf("verify CDK: %v", err)
	}
	if !result.Valid || result.PlanType != "plus" {
		t.Fatalf("unexpected verify result: %+v", result)
	}
}

func TestManagedRechargeUpstreamValidateSessionUsesExpectedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/session/validate" {
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode upstream payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload["platform"] != "chatgpt" || payload["token"] != "test-session" {
			t.Errorf("unexpected validation payload: %+v", payload)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"email":"verified@example.com","subscription":{"plan_type":"plus","has_active_subscription":true}}`))
	}))
	defer server.Close()

	client := &managedRechargeUpstreamClient{baseURL: server.URL, client: server.Client()}
	result, err := client.validateSession(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("validate Session: %v", err)
	}
	if !result.Valid || result.Email != "verified@example.com" || result.Subscription == nil || result.Subscription.PlanType != "plus" {
		t.Fatalf("unexpected validation result: %+v", result)
	}
}

func TestManagedRechargeUpstreamLookupRejectsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"CDK not found"}`))
	}))
	defer server.Close()

	client := &managedRechargeUpstreamClient{baseURL: server.URL, client: server.Client()}
	if _, err := client.lookupTask(context.Background(), "missing-cdk"); err == nil {
		t.Fatal("lookup accepted provider error response")
	}
}

func TestManagedRechargeUpstreamPreservesHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	client := &managedRechargeUpstreamClient{baseURL: server.URL, client: server.Client()}
	_, err := client.lookupTask(context.Background(), "missing-cdk")
	var upstreamError *managedRechargeUpstreamHTTPError
	if !errors.As(err, &upstreamError) || upstreamError.StatusCode != http.StatusNotFound {
		t.Fatalf("lookup error = %v, want HTTP 404 upstream error", err)
	}
}

func TestManagedRechargeUpstreamRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"message":"` + strings.Repeat("x", managedRechargeMaxResponseSize) + `"}`))
	}))
	defer server.Close()

	client := &managedRechargeUpstreamClient{baseURL: server.URL, client: server.Client()}
	var output map[string]any
	err := client.doJSON(context.Background(), http.MethodGet, "/large", nil, &output)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("oversized response error = %v", err)
	}
}
