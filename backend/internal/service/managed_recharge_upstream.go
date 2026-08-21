package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	managedRechargeUpstreamBaseURL = "https://redeem.desolate.codes"
	managedRechargeMaxResponseSize = 1 << 20
)

type managedRechargeUpstreamClient struct {
	baseURL string
	client  *http.Client
}

type managedRechargeVerifyResponse struct {
	Valid          bool   `json:"valid"`
	PlanType       string `json:"plan_type"`
	PlanName       string `json:"plan_name"`
	ProcessingMode string `json:"processing_mode"`
	Error          string `json:"error"`
	Message        string `json:"message"`
}

type managedRechargeCreateResponse struct {
	TaskID  string `json:"task_id"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

type managedRechargeConfirmResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

type managedRechargeLookupResponse struct {
	TaskID             string `json:"task_id"`
	TaskStatus         string `json:"task_status"`
	AccountEmail       string `json:"account_email"`
	FailureReason      string `json:"failure_reason"`
	PostProcessStatus  string `json:"post_process_status"`
	PostProcessCode    string `json:"post_process_code"`
	PostProcessUpdated string `json:"post_process_updated_at"`
	QueuePosition      int    `json:"queue_position"`
	QueueTotal         int    `json:"queue_total"`
	Progress           string `json:"progress"`
	Error              string `json:"error"`
	Message            string `json:"message"`
}

type managedRechargeReplacementSessionResponse struct {
	PostProcessStatus string `json:"post_process_status"`
	PostProcessCode   string `json:"post_process_code"`
	Error             string `json:"error"`
	Message           string `json:"message"`
}

func newManagedRechargeUpstreamClient() *managedRechargeUpstreamClient {
	return &managedRechargeUpstreamClient{
		baseURL: managedRechargeUpstreamBaseURL,
		client: &http.Client{
			Timeout: 35 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *managedRechargeUpstreamClient) verifyCDK(ctx context.Context, code string) (*managedRechargeVerifyResponse, error) {
	var result managedRechargeVerifyResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/recharge/verify-cdk", map[string]string{
		"cdk_code": code,
	}, &result)
	return &result, err
}

func (c *managedRechargeUpstreamClient) createTask(ctx context.Context, code, session string) (*managedRechargeCreateResponse, error) {
	var result managedRechargeCreateResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/recharge/create-task", map[string]string{
		"cdk_code":     code,
		"session_json": session,
	}, &result)
	return &result, err
}

func (c *managedRechargeUpstreamClient) confirmTask(ctx context.Context, taskID string) (*managedRechargeConfirmResponse, error) {
	var result managedRechargeConfirmResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/recharge/confirm-task", map[string]string{
		"task_id": taskID,
	}, &result)
	return &result, err
}

func (c *managedRechargeUpstreamClient) lookupTask(ctx context.Context, code string) (*managedRechargeLookupResponse, error) {
	var result managedRechargeLookupResponse
	path := "/api/v1/lookup/task?cdk_code=" + url.QueryEscape(code)
	err := c.doJSON(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (c *managedRechargeUpstreamClient) submitReplacementSession(ctx context.Context, code, session string) (*managedRechargeReplacementSessionResponse, error) {
	var result managedRechargeReplacementSessionResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/lookup/task/session", map[string]string{
		"cdk_code":     code,
		"session_json": session,
	}, &result)
	return &result, err
}

func (c *managedRechargeUpstreamClient) doJSON(ctx context.Context, method, path string, input any, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode managed recharge upstream request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create managed recharge upstream request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ToCreate-Managed-Recharge/1.0")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("managed recharge upstream request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, managedRechargeMaxResponseSize+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read managed recharge upstream response: %w", err)
	}
	if len(payload) > managedRechargeMaxResponseSize {
		return fmt.Errorf("managed recharge upstream response exceeds limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("managed recharge upstream returned status %d", resp.StatusCode)
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return fmt.Errorf("managed recharge upstream returned an empty response")
	}
	if err := json.Unmarshal(payload, output); err != nil {
		return fmt.Errorf("decode managed recharge upstream response: %w", err)
	}
	return nil
}
