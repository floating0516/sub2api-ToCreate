package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newCustomUpdateTestRouter(handler *CustomBuildHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/status", handler.GetCustomUpdateStatus)
	router.GET("/resolver-config", handler.GetCustomUpdateResolverConfig)
	router.PUT("/resolver-config", handler.UpdateCustomUpdateResolverConfig)
	router.POST("/resolver-config/test", handler.TestCustomUpdateResolverConfig)
	router.POST("/stage", handler.StartCustomUpdate)
	router.POST("/resolution/accept", handler.AcceptCustomUpdateResolution)
	router.POST("/resolution/abort", handler.AbortCustomUpdateResolution)
	router.POST("/promote", handler.PromoteCustomUpdate)
	return router
}

func newCustomUpdateTestHandler(t *testing.T) (*CustomBuildHandler, string) {
	t.Helper()
	controlDir := t.TempDir()
	handler := &CustomBuildHandler{updateControlDir: controlDir}
	return handler, controlDir
}

func markCustomUpdateControllerOnline(t *testing.T, controlDir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(controlDir, customUpdateHeartbeatFile),
		[]byte(time.Now().UTC().Format(time.RFC3339)),
		0644,
	))
}

func writeCustomUpdateTestStatus(t *testing.T, controlDir string, status customUpdateStatus) {
	t.Helper()
	data, err := json.Marshal(status)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(controlDir, customUpdateStatusFile), data, 0644))
}

func TestCustomUpdateStatusReportsOfflineController(t *testing.T) {
	handler, _ := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"enabled":true`)
	require.Contains(t, recorder.Body.String(), `"controller_online":false`)
	require.Contains(t, recorder.Body.String(), `"state":"idle"`)
}

func TestCustomUpdateStatusReturnsControllerSteps(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	writeCustomUpdateTestStatus(t, controlDir, customUpdateStatus{
		State:        "resolution_ready",
		ResolutionID: "0123456789abcdef0123456789abcdef",
		ConflictFiles: []string{
			"frontend/src/example.ts",
		},
		ResolutionRiskLevel: "medium",
		Steps: []customUpdateStep{
			{ID: "source_check", Status: "completed"},
			{ID: "conflict_resolution", Status: "action_required"},
		},
	})
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"id":"source_check","status":"completed"`)
	require.Contains(t, recorder.Body.String(), `"id":"conflict_resolution","status":"action_required"`)
	require.Contains(t, recorder.Body.String(), `"conflict_files":["frontend/src/example.ts"]`)
	require.Contains(t, recorder.Body.String(), `"resolution_risk_level":"medium"`)
}

func TestStartCustomUpdateQueuesOnlyOneFixedStageAction(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stage", nil))
	require.Equal(t, http.StatusAccepted, recorder.Code)

	requestData, err := os.ReadFile(filepath.Join(controlDir, customUpdateRequestFile))
	require.NoError(t, err)
	var request customUpdateRequest
	require.NoError(t, json.Unmarshal(requestData, &request))
	require.Equal(t, "stage", request.Action)
	require.Empty(t, request.Image)
	require.Len(t, request.ID, 32)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stage", nil))
	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestStartCustomUpdateRejectsRequestAlreadyBeingProcessed(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	require.NoError(t, os.WriteFile(
		filepath.Join(controlDir, customUpdateProcessingFile),
		[]byte(`{"id":"0123456789abcdef0123456789abcdef","action":"stage"}`),
		0644,
	))
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stage", nil))

	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestPromoteCustomUpdateRequiresExactStagedImage(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	const stagedImage = "ghcr.io/floating0516/sub2api-tocreate:0.1.164-tc1.17"
	writeCustomUpdateTestStatus(t, controlDir, customUpdateStatus{
		State: "awaiting_approval",
		Image: stagedImage,
	})
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"image":"ghcr.io/floating0516/sub2api-tocreate:wrong"}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/promote", body))
	require.Equal(t, http.StatusConflict, recorder.Code)

	recorder = httptest.NewRecorder()
	body = bytes.NewBufferString(`{"image":"` + stagedImage + `"}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/promote", body))
	require.Equal(t, http.StatusAccepted, recorder.Code)

	requestData, err := os.ReadFile(filepath.Join(controlDir, customUpdateRequestFile))
	require.NoError(t, err)
	var request customUpdateRequest
	require.NoError(t, json.Unmarshal(requestData, &request))
	require.Equal(t, "promote", request.Action)
	require.Equal(t, stagedImage, request.Image)
}

func TestAcceptCustomUpdateResolutionRequiresExactPendingID(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	const resolutionID = "0123456789abcdef0123456789abcdef"
	writeCustomUpdateTestStatus(t, controlDir, customUpdateStatus{
		State:         "resolution_ready",
		ResolutionID:  resolutionID,
		ConflictFiles: []string{"frontend/src/example.ts"},
	})
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"resolution_id":"ffffffffffffffffffffffffffffffff"}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/resolution/accept", body))
	require.Equal(t, http.StatusConflict, recorder.Code)

	recorder = httptest.NewRecorder()
	body = bytes.NewBufferString(`{"resolution_id":"` + resolutionID + `"}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/resolution/accept", body))
	require.Equal(t, http.StatusAccepted, recorder.Code)

	requestData, err := os.ReadFile(filepath.Join(controlDir, customUpdateRequestFile))
	require.NoError(t, err)
	var request customUpdateRequest
	require.NoError(t, json.Unmarshal(requestData, &request))
	require.Equal(t, "accept_resolution", request.Action)
	require.Equal(t, resolutionID, request.ResolutionID)
	require.Empty(t, request.Image)
}

func TestAbortCustomUpdateResolutionAllowsResolverFailure(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	const resolutionID = "abcdef0123456789abcdef0123456789"
	writeCustomUpdateTestStatus(t, controlDir, customUpdateStatus{
		State:        "resolution_failed",
		ResolutionID: resolutionID,
	})
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"resolution_id":"` + resolutionID + `"}`)
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/resolution/abort", body))
	require.Equal(t, http.StatusAccepted, recorder.Code)

	requestData, err := os.ReadFile(filepath.Join(controlDir, customUpdateRequestFile))
	require.NoError(t, err)
	var request customUpdateRequest
	require.NoError(t, json.Unmarshal(requestData, &request))
	require.Equal(t, "abort_resolution", request.Action)
	require.Equal(t, resolutionID, request.ResolutionID)
}

func TestStartCustomUpdateRejectsOfflineController(t *testing.T) {
	handler, _ := newCustomUpdateTestHandler(t)
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stage", nil))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestStartCustomUpdateRejectsWhileSourcePushIsActive(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	writeCustomUpdateTestStatus(t, controlDir, customUpdateStatus{State: "pushing"})
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stage", nil))

	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestStartCustomUpdateRejectsWhileResolutionAwaitsReview(t *testing.T) {
	handler, controlDir := newCustomUpdateTestHandler(t)
	markCustomUpdateControllerOnline(t, controlDir)
	writeCustomUpdateTestStatus(t, controlDir, customUpdateStatus{State: "resolution_ready"})
	router := newCustomUpdateTestRouter(handler)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/stage", nil))

	require.Equal(t, http.StatusConflict, recorder.Code)
}
