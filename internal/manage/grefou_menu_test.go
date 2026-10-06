package manage

import (
	"github.com/backpack/backpack/internal/tui"
	"strings"
	"testing"
)

func TestGREFOUSetupFamilyAndDirection(t *testing.T) {
	restore := tui.SetInput(strings.NewReader("4\n1\n"))
	defer restore()
	var got string
	out := capture(t, func() { got = chooseTransport(true) })
	if got != "gre-fou" || !strings.Contains(out, "Select GRE Transport") {
		t.Fatalf("GRE family selection: %q\n%s", got, out)
	}
	for _, tc := range []struct {
		side      directSide
		direction tunnelDirection
		dial      bool
	}{
		{sideIran, directionReverse, false}, {sideKharej, directionReverse, true},
		{sideIran, directionDirect, true}, {sideKharej, directionDirect, false},
	} {
		if greFOUDials(tc.side, tc.direction) != tc.dial {
			t.Fatalf("wrong initiation for %v/%v", tc.side, tc.direction)
		}
	}
}

func TestGREFOUIsNotAStreamConversionTarget(t *testing.T) {
	restore := tui.SetInput(strings.NewReader("4\n0\n"))
	defer restore()
	out := capture(t, func() {
		if chooseTransport() != "" {
			t.Error("unexpected transport")
		}
	})
	if strings.Contains(out, "GRE over FOU") {
		t.Fatal("L3 offered as a stream conversion")
	}
}

func TestDirectionMenuHasOnlyDirections(t *testing.T) {
	restore := tui.SetInput(strings.NewReader("2\n"))
	defer restore()
	out := capture(t, func() {
		if askDirection("Iran") != directionDirect {
			t.Error("wrong direction")
		}
	})
	if strings.Contains(out, "GRE over FOU") {
		t.Fatal("carrier leaked into direction menu")
	}
}
