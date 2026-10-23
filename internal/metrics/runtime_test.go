package metrics

import (
	"os"
	"os/exec"
	"sync"
	"testing"
)

// The counters have to move when the thing they count moves, or they are
// decoration. Both leaks this codebase has had were a number climbing and not
// coming back down; a counter that does not track its subject would have shown
// a flat line through both.
func TestTheRuntimeCountersTrackWhatTheyCount(t *testing.T) {
	// The count is process-wide. A collector from an earlier test can finish
	// between the baseline and the second sample, hiding one of the goroutines
	// held here. Use a fresh test process so the same assertion measures only
	// this workload, including when the race detector is enabled.
	if os.Getenv("BACKPACK_METRICS_RUNTIME_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestTheRuntimeCountersTrackWhatTheyCount$", "-test.count=1")
		cmd.Env = append(os.Environ(), "BACKPACK_METRICS_RUNTIME_CHILD=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated runtime counter check failed: %v\n%s", err, output)
		}
		return
	}
	before := readRuntimeStats()
	if before.Goroutines < 1 {
		t.Fatalf("goroutine count came back as %d, which cannot be right", before.Goroutines)
	}

	// Hold a known number of goroutines and see the count move by at least
	// that much. "At least", not "exactly": the test binary and the runtime
	// have goroutines of their own that come and go.
	const held = 50
	release := make(chan struct{})
	var started, finished sync.WaitGroup
	started.Add(held)
	finished.Add(held)
	for i := 0; i < held; i++ {
		go func() { defer finished.Done(); started.Done(); <-release }()
	}
	started.Wait()

	during := readRuntimeStats()
	if during.Goroutines < before.Goroutines+held {
		t.Errorf("held %d extra goroutines but the count went %d -> %d; it is not tracking them",
			held, before.Goroutines, during.Goroutines)
	}

	close(release)
	finished.Wait()

	// And the heap figures are real numbers rather than zero.
	if during.HeapBytes == 0 || during.HeapObjects == 0 {
		t.Errorf("heap figures came back as %d bytes / %d objects; something is not being read",
			during.HeapBytes, during.HeapObjects)
	}
}

// The descriptor count is a diagnostic, so it degrades rather than failing: a
// platform without /proc/self/fd reports nothing instead of breaking the
// snapshot. Where it does work it must not count the handle it used to look.
func TestTheOpenFileCountIsHonestOrAbsent(t *testing.T) {
	n := openFileCount()
	if n == 0 {
		t.Skip("this platform does not expose /proc/self/fd")
	}

	// Opening a file must move it by one.
	f, err := openTempFile(t)
	if err != nil {
		t.Skipf("cannot open a file here: %v", err)
	}
	after := openFileCount()
	if after != n+1 {
		t.Errorf("opened one file and the count went %d -> %d, want %d", n, after, n+1)
	}
	f.Close()
}

func openTempFile(t *testing.T) (*os.File, error) {
	t.Helper()
	return os.CreateTemp(t.TempDir(), "fdcount")
}
