package routes

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestInstallScriptsUseRequestOriginWithoutEmbeddingSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerInstallScriptRoutes(router)

	tests := []struct {
		path        string
		contentType string
	}{
		{"/install.sh", "text/x-shellscript"},
		{"/install.ps1", "text/plain"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Host = "api.example.com"
			request.Header.Set("X-Forwarded-Proto", "https")
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			require.Contains(t, recorder.Header().Get("Content-Type"), test.contentType)
			require.Contains(t, recorder.Body.String(), "https://api.example.com")
			require.Contains(t, recorder.Body.String(), "/api/v1/install-token/redeem")
			require.Contains(t, recorder.Body.String(), " ______   ______   ______     ______")
			require.Contains(t, recorder.Body.String(), "ToCreate Quick Start")
			require.NotContains(t, recorder.Body.String(), installScriptBaseURLPlaceholder)
			require.NotContains(t, recorder.Body.String(), "sk-test")

			body := recorder.Body.String()
			if test.path == "/install.sh" {
				require.Contains(t, body, "https://deb.nodesource.com/node_${MIN_NODE_VERSION}.x")
				require.Contains(t, body, "https://rpm.nodesource.com/pub_${MIN_NODE_VERSION}.x")
				require.NotContains(t, body, "apt-get install -y nodejs npm")
				require.Contains(t, body, "--headless")
				require.Contains(t, body, `[ -z "${DISPLAY:-}" ]`)
				require.Contains(t, body, `[ -z "${WAYLAND_DISPLAY:-}" ]`)
				require.Contains(t, body, `curl -fsSL "${BASE_URL%/}/install-config.js"`)
				require.Contains(t, body, `node --check "$CONFIG_HELPER"`)
				require.Contains(t, body, `if node "$CONFIG_HELPER"`)
				require.Contains(t, body, `--response "$response_file"`)
				require.Contains(t, body, "The one-click import remains available at:")

				mainIndex := strings.LastIndex(body, "\nmain() {")
				require.NotEqual(t, -1, mainIndex)
				main := body[mainIndex:]
				fetchIndex := strings.Index(main, "fetch_install_metadata")
				ensureNodeIndex := strings.Index(main, "ensure_node")
				downloadHelperIndex := strings.Index(main, "download_config_helper")
				redeemIndex := strings.Index(main, "redeem_and_import")
				require.NotEqual(t, -1, fetchIndex)
				require.NotEqual(t, -1, ensureNodeIndex)
				require.NotEqual(t, -1, downloadHelperIndex)
				require.NotEqual(t, -1, redeemIndex)
				require.Less(t, fetchIndex, ensureNodeIndex)
				require.Less(t, downloadHelperIndex, redeemIndex)
				require.Contains(t, main, `if [ "$HEADLESS" -eq 1 ]; then`)
				require.Contains(t, main, "else\n    ensure_cc_switch")
			} else {
				require.Contains(t, body, "Invoke-NodeWinget")
				require.Contains(t, body, "OpenJS.NodeJS.LTS")
				require.Contains(t, body, "\"upgrade\"")
				require.Contains(t, body, "\"--force\"")

				preflightIndex := strings.LastIndex(body, "Write-Section \"1. Preflight\"")
				require.NotEqual(t, -1, preflightIndex)
				preflight := body[preflightIndex:]
				loadMetadataIndex := strings.Index(preflight, "Load-InstallMetadata")
				ensureNodeIndex := strings.Index(preflight, "Ensure-Node")
				require.NotEqual(t, -1, loadMetadataIndex)
				require.NotEqual(t, -1, ensureNodeIndex)
				require.Less(t, loadMetadataIndex, ensureNodeIndex)
			}
		})
	}
}

func TestInstallConfigHelperRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerInstallScriptRoutes(router)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/install-config.js", nil)
	request.Host = "api.example.com"
	request.Header.Set("X-Forwarded-Proto", "https")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/javascript")
	require.Contains(t, recorder.Body.String(), "function configureClaude")
	require.Contains(t, recorder.Body.String(), "function configureCodex")
	require.Contains(t, recorder.Body.String(), "function configureGemini")
	require.NotContains(t, recorder.Body.String(), installScriptBaseURLPlaceholder)
	require.NotContains(t, recorder.Body.String(), "sk-test")
}

func TestInstallConfigHelperNodeSuite(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not available")
	}
	versionOutput, err := exec.Command(node, "-p", `Number(process.versions.node.split(".")[0])`).Output()
	if err != nil {
		t.Skip("Node.js version could not be determined")
	}
	majorVersion, err := strconv.Atoi(strings.TrimSpace(string(versionOutput)))
	if err != nil || majorVersion < 22 {
		t.Skip("Node.js 22 or newer is required")
	}

	cmd := exec.Command(node, "--test", "install_scripts/install-config.test.js")
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "Node helper tests failed:\n%s", output)
}
