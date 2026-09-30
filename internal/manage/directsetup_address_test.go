package manage

import (
	"testing"
)

func TestManualL3AddressesCannotReuseAnExistingTunnelBlock(t *testing.T) {
	// The missing /30 is a supported point-to-point form, but it must not let
	// the operator reuse the same 10.10.0.x block as an existing tunnel.
	if err := checkL3BlockAvailable("10.10.0.1", "existing"); err == nil {
		t.Fatal("an existing tunnel's address block was accepted")
	}
	if err := checkL3BlockAvailable("10.10.1.1/30", ""); err != nil {
		t.Fatalf("a free /30 block was rejected: %v", err)
	}
}
