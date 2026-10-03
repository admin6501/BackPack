package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeSupportHandle(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"@help_desk", "help_desk"},
		{"https://t.me/help_desk", "help_desk"},
		{"", ""},
		{"help", ""},
		{"help desk", ""},
		{"https://example.com/help", ""},
	} {
		if got := normalizeSupportHandle(tc.in); got != tc.want {
			t.Errorf("normalizeSupportHandle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCustomerPortsHideBackendAddresses(t *testing.T) {
	got := customerPorts([]string{"443", "2087=127.0.0.1:11002", "2096=10.0.0.4:2097", "9000-9003"})
	want := []string{"443", "2087 → 11002", "2096 → 2097", "9000-9003"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("customerPorts = %#v, want %#v", got, want)
	}
}

func TestPublicStatusPageRequiresEnabledBearerID(t *testing.T) {
	id := strings.Repeat("a", 48)
	useConfigFile(t, Config{PublicTunnelLinks: map[string]PublicTunnelLink{
		"one": {ID: id, Enabled: true},
		"two": {ID: strings.Repeat("b", 48), Enabled: false},
	}})
	s := &server{}
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/status/" + id, 200},
		{"/status/" + id + "/", 404},
		{"/status/" + strings.Repeat("b", 48), 404},
		{"/status/not-a-token", 404},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		s.handlePublicStatusPage(w, r)
		if w.Code != tc.want {
			t.Errorf("GET %s returned %d, want %d", tc.path, w.Code, tc.want)
		}
		if w.Code == 200 && (strings.Contains(w.Body.String(), "__CSP_NONCE__") || w.Header().Get("Cache-Control") != "no-store") {
			t.Errorf("public page retained nonce placeholder or omitted no-store headers")
		}
	}
}

func TestPublicStatusDataDoesNotAcceptAnUnknownID(t *testing.T) {
	useConfigFile(t, Config{})
	r := httptest.NewRequest("GET", "/api/public/status?id="+strings.Repeat("c", 48), nil)
	w := httptest.NewRecorder()
	(&server{}).handlePublicStatusData(w, r)
	if w.Code != 404 {
		t.Fatalf("unknown public ID returned %d, want 404", w.Code)
	}
}

func TestCustomerRoutesDoNotExposePanelPath(t *testing.T) {
	id := strings.Repeat("a", 48)
	useConfigFile(t, Config{BasePath: "private-admin", PublicTunnelLinks: map[string]PublicTunnelLink{"one": {ID: id, Enabled: true}}})
	panel := withBasePath("/private-admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	handler := (&server{}).withPublicStatusRoutes(panel)
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/status/" + id, 200},
		{"/status/" + id + "/", 404},
		{"/api/public/status?id=unknown", 404},
		{"/api/public/status/", 404},
		{"/api/tunnel/list", 404},
		{"/login", 404},
		{"/", 404},
		{"/private-admin/login", 401},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.want {
			t.Errorf("%s returned %d, want %d", tc.path, w.Code, tc.want)
		}
		if tc.want == 200 {
			if strings.Contains(w.Body.String(), "private-admin") {
				t.Error("customer page leaks administrator path")
			}
			if w.Header().Get("Content-Security-Policy") == "" {
				t.Error("customer route lost security headers")
			}
		}
	}
	if got := publicURL(httptest.NewRequest("GET", "http://example.com/private-admin/api/tunnel/public-link", nil), id); got != "http://example.com/status/"+id {
		t.Fatalf("publicURL leaked panel prefix: %s", got)
	}
}
