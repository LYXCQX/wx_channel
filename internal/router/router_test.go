package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wx_channel/internal/config"
	"wx_channel/internal/websocket"

	"github.com/qtgolang/SunnyNet/SunnyNet"
)

func newTestRouter() *APIRouter {
	cfg := &config.Config{
		Port:           2025,
		AllowedOrigins: []string{"*"},
	}
	hub := websocket.NewHub()
	sunny := SunnyNet.NewSunny()

	// Create router with nil dependencies where possible or mocked ones
	r := NewAPIRouter(cfg, hub, sunny)
	return r
}

func TestSystemAPI(t *testing.T) {
	router := newTestRouter()

	// Test Info
	req, _ := http.NewRequest("GET", "/api/v1/system/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", resp["code"])
	}

	// Test Health
	req, _ = http.NewRequest("GET", "/api/v1/system/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLogsAPI(t *testing.T) {
	router := newTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/logs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestProxyAPI(t *testing.T) {
	router := newTestRouter()

	// Test Status
	req, _ := http.NewRequest("GET", "/api/v1/proxy/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSearchAPI(t *testing.T) {
	router := newTestRouter()

	// Test Contact Search (expecting 500 because no WS client connected)
	params := map[string]interface{}{
		"keyword":   "test",
		"page":      1,
		"page_size": 10,
	}
	body, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/v1/search/contact", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expecting 503 Service Unavailable (no available client)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (no client), got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if msg, ok := resp["message"].(string); !ok || !strings.Contains(msg, "WeChat client not connected") {
		t.Logf("Got message: %v", msg)
	}

	// Test Feed Search
	req, _ = http.NewRequest("GET", "/api/v1/search/feed?username=test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestCertificateAPI(t *testing.T) {
	router := newTestRouter()

	// Test Status
	req, _ := http.NewRequest("GET", "/api/v1/certificate/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Note: Verify response format. CheckCertificate might fail or satisfy depending on environment
	// We just check if it returns a valid JSON response structure
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to parse response JSON: %v", err)
	}
}

func TestAuthMiddleware_PublicPathsBypass(t *testing.T) {
	handler := AuthMiddleware("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	publicPaths := []string{
		"/api/health",
		"/api/console/verify-token",
		"/api/system/health",
		"/api/v1/system/health",
	}

	for _, p := range publicPaths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected public path %s to bypass auth, got %d", p, w.Code)
		}
	}
}

func TestAuthMiddleware_ProtectedPathRequiresToken(t *testing.T) {
	handler := AuthMiddleware("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 无 token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}

	// 正确 token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	req.Header.Set("X-Local-Auth", "secret-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", w.Code)
	}
}

func TestVideoPlayRoute_UsesPlayHandler(t *testing.T) {
	router := newTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/video/play", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	msg, _ := resp["message"].(string)
	if msg == "" {
		msg, _ = resp["error"].(string)
	}
	if !strings.Contains(msg, "url parameter is required") {
		t.Fatalf("expected play handler error message, got: %v", msg)
	}
}
