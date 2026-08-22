package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	managedRechargeProviderModeEnv        = "MANAGED_RECHARGE_PROVIDER_MODE"
	managedRechargeEnvironmentEnv         = "MANAGED_RECHARGE_ENVIRONMENT"
	managedRechargeMockStepSecondsEnv     = "MANAGED_RECHARGE_MOCK_STEP_SECONDS"
	managedRechargeMockEnvironment        = "staging"
	managedRechargeDefaultMockStepSeconds = 10
	managedRechargeMaxMockStepSeconds     = 300

	managedRechargeMockSessionEmail = "mock@example.com"
	managedRechargeMockAccessToken  = "mock-access-token"
)

var managedRechargeMockCDKPattern = regexp.MustCompile(`^MOCK-(PLUS-SUCCESS|PRO-SUCCESS|SESSION-REQUIRED|FAIL-REFUND)(-[0-9]{3})?$`)

type managedRechargeMockScenario string

const (
	managedRechargeMockPlusSuccess    managedRechargeMockScenario = "PLUS-SUCCESS"
	managedRechargeMockProSuccess     managedRechargeMockScenario = "PRO-SUCCESS"
	managedRechargeMockSessionNeeded  managedRechargeMockScenario = "SESSION-REQUIRED"
	managedRechargeMockFailureRefund  managedRechargeMockScenario = "FAIL-REFUND"
)

type managedRechargeMockTask struct {
	taskID                 string
	scenario               managedRechargeMockScenario
	createdAt              time.Time
	replacementSubmittedAt time.Time
}

type managedRechargeMockUpstream struct {
	mu     sync.Mutex
	step   time.Duration
	now    func() time.Time
	byCode map[string]*managedRechargeMockTask
	byID   map[string]*managedRechargeMockTask
}

func newManagedRechargeUpstreamFromEnvironment() (managedRechargeUpstream, bool, int, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(managedRechargeProviderModeEnv)))
	switch mode {
	case "", "real":
		return newManagedRechargeUpstreamClient(), false, 0, nil
	case "mock":
		if environment := strings.ToLower(strings.TrimSpace(os.Getenv(managedRechargeEnvironmentEnv))); environment != managedRechargeMockEnvironment {
			return nil, false, 0, fmt.Errorf("%s=mock requires %s=%s", managedRechargeProviderModeEnv, managedRechargeEnvironmentEnv, managedRechargeMockEnvironment)
		}
		stepSeconds, err := managedRechargeMockStepSecondsFromEnvironment()
		if err != nil {
			return nil, false, 0, err
		}
		return newManagedRechargeMockUpstream(time.Duration(stepSeconds)*time.Second), true, stepSeconds, nil
	default:
		return nil, false, 0, fmt.Errorf("unsupported %s value %q", managedRechargeProviderModeEnv, mode)
	}
}

func managedRechargeMockStepSecondsFromEnvironment() (int, error) {
	raw := strings.TrimSpace(os.Getenv(managedRechargeMockStepSecondsEnv))
	if raw == "" {
		return managedRechargeDefaultMockStepSeconds, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 1 || seconds > managedRechargeMaxMockStepSeconds {
		return 0, fmt.Errorf("%s must be between 1 and %d", managedRechargeMockStepSecondsEnv, managedRechargeMaxMockStepSeconds)
	}
	return seconds, nil
}

func newManagedRechargeMockUpstream(step time.Duration) *managedRechargeMockUpstream {
	return &managedRechargeMockUpstream{
		step:   step,
		now:    time.Now,
		byCode: make(map[string]*managedRechargeMockTask),
		byID:   make(map[string]*managedRechargeMockTask),
	}
}

func (m *managedRechargeMockUpstream) verifyCDK(_ context.Context, code string) (*managedRechargeVerifyResponse, error) {
	scenario, ok := managedRechargeMockScenarioForCDK(code)
	if !ok {
		return &managedRechargeVerifyResponse{Valid: false, Error: "mock_cdk_invalid"}, nil
	}
	planType := "plus"
	if scenario == managedRechargeMockProSuccess {
		planType = "pro"
	}
	return &managedRechargeVerifyResponse{
		Valid:          true,
		PlanType:       planType,
		PlanName:       "Mock " + strings.ToUpper(planType),
		ProcessingMode: "mock",
	}, nil
}

func (m *managedRechargeMockUpstream) createTask(_ context.Context, code, session string) (*managedRechargeCreateResponse, error) {
	scenario, ok := managedRechargeMockScenarioForCDK(code)
	if !ok {
		return &managedRechargeCreateResponse{Error: "mock_cdk_invalid"}, nil
	}
	if !isManagedRechargeMockSession(session) {
		return &managedRechargeCreateResponse{Error: "mock_session_required"}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing := m.byCode[code]; existing != nil {
		return &managedRechargeCreateResponse{TaskID: existing.taskID}, nil
	}
	taskID := fmt.Sprintf("mock-%s-%d", strings.ToLower(string(scenario)), m.now().UnixNano())
	task := &managedRechargeMockTask{
		taskID:    taskID,
		scenario:  scenario,
		createdAt: m.now(),
	}
	m.byCode[code] = task
	m.byID[taskID] = task
	return &managedRechargeCreateResponse{TaskID: taskID}, nil
}

func (m *managedRechargeMockUpstream) confirmTask(_ context.Context, taskID string) (*managedRechargeConfirmResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.byID[taskID] == nil {
		return nil, &managedRechargeUpstreamHTTPError{StatusCode: http.StatusNotFound}
	}
	return &managedRechargeConfirmResponse{Status: "queued"}, nil
}

func (m *managedRechargeMockUpstream) lookupTask(_ context.Context, code string) (*managedRechargeLookupResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task := m.byCode[code]
	if task == nil {
		return nil, &managedRechargeUpstreamHTTPError{StatusCode: http.StatusNotFound}
	}

	result := &managedRechargeLookupResponse{
		TaskID:       task.taskID,
		AccountEmail: managedRechargeMockSessionEmail,
		QueueTotal:   4,
	}
	phase := managedRechargeMockPhase(m.now().Sub(task.createdAt), m.step)
	if phase == 0 {
		result.TaskStatus = "queued"
		result.QueuePosition = 2
		result.Progress = "模拟任务已进入队列"
		return result, nil
	}
	if phase == 1 {
		result.TaskStatus = "processing"
		result.Progress = "模拟渠道正在处理"
		return result, nil
	}

	switch task.scenario {
	case managedRechargeMockFailureRefund:
		result.TaskStatus = "failed"
		result.FailureReason = "session_invalid"
		result.Progress = "模拟履约失败"
	case managedRechargeMockSessionNeeded:
		result.TaskStatus = "completed"
		if task.replacementSubmittedAt.IsZero() {
			result.PostProcessStatus = "action_required"
			result.PostProcessCode = "replacement_session_required"
			result.Progress = "等待补交模拟 Session"
		} else if m.now().Sub(task.replacementSubmittedAt) < m.step {
			result.PostProcessStatus = "processing"
			result.Progress = "正在验证补交的模拟 Session"
		} else {
			result.PostProcessStatus = "completed"
			result.Progress = "模拟充值完成"
		}
	case managedRechargeMockPlusSuccess, managedRechargeMockProSuccess:
		result.TaskStatus = "completed"
		if phase == 2 {
			result.PostProcessStatus = "processing"
			result.Progress = "正在确认模拟订阅"
		} else {
			result.PostProcessStatus = "completed"
			result.Progress = "模拟充值完成"
		}
	}
	return result, nil
}

func (m *managedRechargeMockUpstream) submitReplacementSession(_ context.Context, code, session string) (*managedRechargeReplacementSessionResponse, error) {
	if !isManagedRechargeMockSession(session) {
		return &managedRechargeReplacementSessionResponse{Error: "mock_session_required"}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	task := m.byCode[code]
	if task == nil {
		return nil, &managedRechargeUpstreamHTTPError{StatusCode: http.StatusNotFound}
	}
	if task.scenario != managedRechargeMockSessionNeeded {
		return &managedRechargeReplacementSessionResponse{Error: "replacement_not_required"}, nil
	}
	task.replacementSubmittedAt = m.now()
	return &managedRechargeReplacementSessionResponse{PostProcessStatus: "processing"}, nil
}

func managedRechargeMockScenarioForCDK(code string) (managedRechargeMockScenario, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	matches := managedRechargeMockCDKPattern.FindStringSubmatch(normalized)
	if len(matches) < 2 {
		return "", false
	}
	return managedRechargeMockScenario(matches[1]), true
}

func managedRechargeMockPhase(elapsed, step time.Duration) int {
	if step <= 0 || elapsed <= 0 {
		return 0
	}
	return int(elapsed / step)
}
