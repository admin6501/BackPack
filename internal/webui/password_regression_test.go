package webui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPasswordRouteRequiresAdmin(t *testing.T) {
	isolateAccess(t)
	for _, scope := range []Scope{ScopeRead, ScopeWrite, ScopeAdmin} {
		secret, _, err := IssueToken(fmt.Sprint(scope), scope, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		s := &server{sessions: newSessionStore()}
		mux := http.NewServeMux()
		s.registerCredentialRoutes(mux)
		for _, route := range []string{"password", "backup/export", "backup/import", "restorepoints"} {
			w := httptest.NewRecorder()
			// Unsupported method exercises authorization without exporting or
			// restoring any real machine data if a guard regresses.
			mux.ServeHTTP(w, req("OPTIONS", "/api/"+route, secret))
			want := http.StatusForbidden
			if scope == ScopeAdmin {
				want = http.StatusMethodNotAllowed
				if route == "restorepoints" {
					want = http.StatusOK
				}
			}
			if w.Code != want {
				t.Fatalf("%s scope=%s status=%d want=%d", route, scope, w.Code, want)
			}
		}
	}
}
func TestPasswordAcceptsPanelJSONAndLegacyForm(t *testing.T) {
	old := ConfigPath
	ConfigPath = filepath.Join(t.TempDir(), "webui.json")
	t.Cleanup(func() { ConfigPath = old })
	for _, tc := range []struct{ content, body string }{
		{"application/json", `{"password":"new-password"}`},
		{"application/x-www-form-urlencoded", "password=new-password"},
	} {
		s := &server{sessions: newSessionStore()}
		tok := s.sessions.create("127.0.0.1")
		r := httptest.NewRequest("POST", "/api/password", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", tc.content)
		w := httptest.NewRecorder()
		s.handlePassword(w, r)
		if w.Code != http.StatusOK || Load().Password != "new-password" {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if s.sessions.valid(tok) {
			t.Fatal("old session survived password reset")
		}
	}
	r := httptest.NewRequest("POST", "/api/password", strings.NewReader(`{`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&server{}).handlePassword(w, r)
	if w.Code != 400 || Load().Password != "new-password" {
		t.Fatal("malformed JSON changed credentials")
	}
}
