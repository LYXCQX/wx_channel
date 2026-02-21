package ws

import (
	"net/http/httptest"
	"testing"
)

func TestIsHubOriginAllowed_DefaultPolicy(t *testing.T) {
	t.Setenv("HUB_ALLOWED_ORIGINS", "")

	if !isHubOriginAllowed("") {
		t.Fatalf("expected empty origin to be allowed for non-browser clients")
	}
	if !isHubOriginAllowed("http://localhost:3000") {
		t.Fatalf("expected localhost origin to be allowed")
	}
	if isHubOriginAllowed("https://evil.example.com") {
		t.Fatalf("expected non-local origin to be blocked")
	}
}

func TestIsHubOriginAllowed_ConfiguredPolicy(t *testing.T) {
	t.Setenv("HUB_ALLOWED_ORIGINS", "https://console.example.com,https://admin.example.com")

	if isHubOriginAllowed("") {
		t.Fatalf("expected empty origin to be blocked when policy is configured")
	}
	if !isHubOriginAllowed("https://console.example.com") {
		t.Fatalf("expected listed origin to be allowed")
	}
	if isHubOriginAllowed("https://evil.example.com") {
		t.Fatalf("expected non-listed origin to be blocked")
	}
}

func TestExtractHubToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws/client?token=query-token", nil)
	req.Header.Set("X-Local-Auth", "header-token")
	if got := extractHubToken(req); got != "header-token" {
		t.Fatalf("unexpected token from X-Local-Auth: %q", got)
	}

	req2 := httptest.NewRequest("GET", "/ws/client?token=query-token", nil)
	req2.Header.Set("Authorization", "Bearer bearer-token")
	if got := extractHubToken(req2); got != "bearer-token" {
		t.Fatalf("unexpected token from Authorization: %q", got)
	}

	req3 := httptest.NewRequest("GET", "/ws/client?token=query-token", nil)
	if got := extractHubToken(req3); got != "query-token" {
		t.Fatalf("unexpected token from query: %q", got)
	}
}
