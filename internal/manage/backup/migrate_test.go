package backup

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRestorePausesAllWritersAndResumesOnFailure(t *testing.T) {
	for _, failStop := range []bool{false, true} {
		var calls []string
		run := func(args ...string) (string, error) {
			calls = append(calls, strings.Join(args, " "))
			if args[0] == "is-active" {
				return "active", nil
			}
			if failStop && args[0] == "stop" && args[1] == "panel" {
				return "", errors.New("stop failed")
			}
			return "", nil
		}
		resume, err := pauseRestoreWriters([]string{"monitor", "panel", "tunnel"}, run)
		if (err != nil) != failStop {
			t.Fatalf("pause: %v", err)
		}
		if failed := resume(); len(failed) != 0 {
			t.Fatal(failed)
		}
		if calls[len(calls)-1] != "start monitor" {
			t.Fatal("monitor resumed before tunnel writers")
		}
		if failStop && strings.Contains(strings.Join(calls, ";"), "stop tunnel") {
			t.Fatal("continued after stop failure")
		}
	}
	resume, err := pauseRestoreWriters([]string{"inactive"}, func(args ...string) (string, error) {
		if args[0] == "is-active" {
			return "inactive", errors.New("exit 3")
		}
		return "", fmt.Errorf("must not start")
	})
	if err != nil || len(resume()) != 0 {
		t.Fatal("started a previously inactive service")
	}
	resume, err = pauseRestoreWriters([]string{"panel"}, func(args ...string) (string, error) {
		if args[0] == "is-active" {
			return "active", nil
		}
		if args[0] == "start" {
			return "", errors.New("start failed")
		}
		return "", nil
	})
	if err != nil || len(resume()) != 1 {
		t.Fatal("resume failure was hidden")
	}
}
