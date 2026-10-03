package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomerStatusRejectsLegacyPanelURLs(t *testing.T) {
	id := strings.Repeat("a", 48)
	useConfigFile(t, Config{Password: "12345678", BasePath: "private-admin", PublicTunnelLinks: map[string]PublicTunnelLink{
		"one": {ID: id, Enabled: true},
	}})
	s := &server{}
	for _, prefix := range []string{"/private-admin", ""} {
		h := s.withPublicStatusRoutes(withBasePath(prefix, withPanelSecurity(s.routes())))
		for _, tc := range []struct {
			method, path string
			want         int
		}{
			{"GET", "/status/" + id, http.StatusOK},
			{"POST", "/api/public/status", http.StatusMethodNotAllowed},
			{"GET", "/private-admin/status/" + id, http.StatusNotFound},
			{"GET", "/private-admin/api/public/status?id=" + id, http.StatusNotFound},
			{"POST", "/private-admin/api/public/status", http.StatusNotFound},
		} {
			// In root mode the retired path is outside the panel entirely;
			// its catch-all may require login. The old aliases inside a secret
			// prefix must explicitly return 404 rather than redirect to login.
			if prefix == "" && strings.HasPrefix(tc.path, "/private-admin/") {
				continue
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.want {
				t.Errorf("prefix %q: %s %s = %d, want %d", prefix, tc.method, tc.path, w.Code, tc.want)
			}
			if w.Header().Get("Location") != "" || strings.Contains(w.Body.String(), "private-admin") {
				t.Errorf("customer request disclosed the panel prefix: %s", tc.path)
			}
		}
	}
}

func TestCustomerPathMigrationRetiresDisclosedPathOnce(t *testing.T) {
	id := strings.Repeat("a", 48)
	before := Config{Password: "12345678", Port: 8443, BasePath: "disclosed-admin", HTTPS: true,
		TOTPSecret: "test-secret", RecoveryHashes: []string{"hash"}, SupportTelegram: "help_desk",
		PublicTunnelLinks: map[string]PublicTunnelLink{"one": {ID: id, Enabled: true}}}
	useConfigFile(t, before)
	// Opening the CLI may ensure credentials, but must not rotate a live router.
	if cfg, err := EnsurePassword(); err != nil || !cfg.Equal(before) {
		t.Fatalf("CLI credential read changed the panel: %v", err)
	}
	after, err := migrateCustomerStatusPaths(Load())
	if err != nil {
		t.Fatal(err)
	}
	if after.BasePath == before.BasePath || after.BasePath == "" || !validBasePath(after.BasePath) {
		t.Fatal("disclosed panel path was not replaced with a valid secret path")
	}
	expected := before
	expected.BasePath = after.BasePath
	expected.CustomerStatusPathsIsolated = true
	if !after.Equal(expected) || !Load().Equal(expected) {
		t.Fatal("migration changed other credentials/settings or was not persisted")
	}
	for i := 0; i < 3; i++ {
		cfg, err := migrateCustomerStatusPaths(Load())
		if err != nil || !cfg.Equal(expected) {
			t.Fatalf("restart %d rotated an already migrated panel: %v", i, err)
		}
	}
	h := (&server{}).withPublicStatusRoutes(withBasePath(after.PathPrefix(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	for path, want := range map[string]int{
		before.PathPrefix() + "/login":        http.StatusNotFound,
		before.PathPrefix() + "/status/" + id: http.StatusNotFound,
		after.PathPrefix() + "/login":         http.StatusNoContent,
		"/status/" + id:                       http.StatusOK,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != want || w.Header().Get("Location") != "" {
			t.Errorf("after migration %s = %d, want %d without redirect", path, w.Code, want)
		}
	}
	info, err := os.Stat(configPath())
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("migrated credentials are not stored root-only")
	}
}

func TestCustomerPathMigrationPreservesRootAndUnusedPanel(t *testing.T) {
	for _, tc := range []Config{
		{Password: "12345678", BasePath: "unused-admin"},
		{Password: "12345678", BasePath: "/", PublicTunnelLinks: map[string]PublicTunnelLink{"one": {ID: strings.Repeat("a", 48), Enabled: true}}},
	} {
		useConfigFile(t, tc)
		cfg, err := migrateCustomerStatusPaths(Load())
		if err != nil || cfg.BasePath != tc.BasePath || !cfg.CustomerStatusPathsIsolated {
			t.Fatalf("migration moved an unused/root panel: %v", err)
		}
	}
}

func TestCustomerPathMigrationAlsoRetiresDisabledLinks(t *testing.T) {
	useConfigFile(t, Config{Password: "12345678", BasePath: "disclosed-admin", PublicTunnelLinks: map[string]PublicTunnelLink{
		"disabled": {ID: strings.Repeat("a", 48), Enabled: false},
	}})
	cfg, err := migrateCustomerStatusPaths(Load())
	if err != nil || cfg.BasePath == "disclosed-admin" || cfg.PublicTunnelLinks["disabled"].Enabled {
		t.Fatalf("disabled old link did not retire its disclosed path: %v", err)
	}
}

func TestCustomerPathMigrationDoesNotCompleteOnSaveFailure(t *testing.T) {
	before := Config{Password: "12345678", BasePath: "disclosed-admin", PublicTunnelLinks: map[string]PublicTunnelLink{
		"one": {ID: strings.Repeat("a", 48), Enabled: true},
	}}
	useConfigFile(t, before)
	// A regular file cannot be used as the parent directory of webui.json.
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	configOverride = filepath.Join(blocker, "webui.json")
	after, err := migrateCustomerStatusPaths(before)
	if err == nil || !after.Equal(before) {
		t.Fatal("failed persistence was reported as a completed migration")
	}
}
