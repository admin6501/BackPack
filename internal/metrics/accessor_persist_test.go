package metrics

import "testing"

func TestAccessorTrafficCarriesHistoryAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	first := NewCollector(dir, "l3", "l3-udp", "server", func() uint64 { return 700 }, func() uint64 { return 300 })
	if err := first.Write(); err != nil {
		t.Fatal(err)
	}
	next := NewCollector(dir, "l3", "l3-udp", "server", func() uint64 { return 11 }, func() uint64 { return 13 })
	s := next.Snapshot()
	if s.BytesIn != 711 || s.BytesOut != 313 {
		t.Fatalf("L3 traffic after restart = %d in, %d out", s.BytesIn, s.BytesOut)
	}
}

func TestCounterTrafficDoesNotDoubleOnInProcessReload(t *testing.T) {
	resetCounters(t)
	dir := t.TempDir()
	first := NewCollector(dir, "reverse", "tcp", "server", nil, nil)
	AddBytes(600, 400)
	if err := first.Write(); err != nil {
		t.Fatal(err)
	}
	second := NewCollector(dir, "reverse", "tcp", "server", nil, nil)
	if s := second.Snapshot(); s.BytesIn != 600 || s.BytesOut != 400 {
		t.Fatalf("reloaded traffic doubled: %d in, %d out", s.BytesIn, s.BytesOut)
	}
	AddBytes(12, 7)
	if s := second.Snapshot(); s.BytesIn != 612 || s.BytesOut != 407 {
		t.Fatalf("post-reload traffic = %d in, %d out", s.BytesIn, s.BytesOut)
	}
}
