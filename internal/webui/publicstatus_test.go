package webui

import (
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
