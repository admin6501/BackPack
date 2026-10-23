package manage

import "testing"

// Going back must go back.
//
// The layer-3 editor offers two extra entries on the Iran side, so on the
// kharej side every index is two lower and the switch below it compensates by
// adding two. ChooseOpt answers -1 for "go back", and -1 + 2 is 1 — which is
// the UDP toggle. Pressing 0 to leave would instead ask whether to carry UDP,
// write the config and restart the tunnel.
//
// This models the arithmetic rather than driving the prompt, which needs a
// terminal. l3EditAction is the mapping the editor performs; the editor calls
// it so the two cannot drift.
func TestGoingBackFromTheLayer3EditorGoesBack(t *testing.T) {
	for _, side := range []struct {
		name string
		iran bool
	}{{"iran", true}, {"kharej", false}} {
		if action, ok := l3EditAction(-1, side.iran); ok {
			t.Errorf("%s: going back was taken as action %d", side.name, action)
		}
	}
}

// And the entries themselves must still land where they say they do.
func TestLayer3EditorEntriesMapToTheRightActions(t *testing.T) {
	// Iran sees the shared entries after ports and UDP; Spoof is last when present.
	for chosen, want := range map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6} {
		got, ok := l3EditAction(chosen, true)
		if !ok || got != want {
			t.Errorf("iran: entry %d -> action %d (ok=%v), want %d", chosen, got, ok, want)
		}
	}
	// Kharej sees MTU, the segment cap, token, FEC and optionally Spoof.
	for chosen, want := range map[int]int{0: 2, 1: 3, 2: 4, 3: 5, 4: 6} {
		got, ok := l3EditAction(chosen, false)
		if !ok || got != want {
			t.Errorf("kharej: entry %d -> action %d (ok=%v), want %d", chosen, got, ok, want)
		}
	}
}
