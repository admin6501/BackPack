package localproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPProxyRefusesLocalDestinationsForConnectAndForward(t *testing.T) {
	for _, method := range []string{http.MethodConnect, http.MethodGet} {
		r := httptest.NewRequest(method, "http://169.254.169.254:80/", nil)
		w := httptest.NewRecorder()
		(&httpProxy{}).ServeHTTP(w, r)
		if w.Code != http.StatusBadGateway {
			t.Errorf("%s: status %d, want 502 before accessing metadata", method, w.Code)
		}
	}
}
