package manage

import (
	"strings"
	"testing"

	"github.com/backpack/backpack/config"
)

// The shared-backend editor must retain the fork's tunnel identity and quota.
func TestL3SharedEditRetainsIdentity(t *testing.T) {
	spec := l3SpecOf(Tunnel{Name: "shared-edit-test"}, config.L3Config{
		Mode: "listen", Carrier: "udp", LocalIP: "10.10.1.2/30",
		PeerIP: "10.10.1.1", Addr: "0.0.0.0:9000", Token: "test-token",
	})
	if spec.Name != "shared-edit-test" || spec.Side != sideKharej {
		t.Fatalf("edit lost tunnel identity: name=%q side=%v", spec.Name, spec.Side)
	}
	if !strings.Contains(spec.render(), "[l3]") {
		t.Fatal("edit did not render an L3 tunnel")
	}
}
