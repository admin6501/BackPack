package l3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// FEC headers arrive before Noise authenticates their payloads. A wrong shard
// kind/index must neither crash the reader nor occupy a legitimate data slot.
func TestMalformedFECKindAndIndexDoNotInterruptTheNextPacket(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  byte
		index byte
	}{
		{"data in first parity slot", fecKindData, 4},
		{"data in last parity slot", fecKindData, 5},
		{"parity in first data slot", fecKindParity, 0},
		{"parity in last data slot", fecKindParity, 3},
		{"data beyond group", fecKindData, 6},
		{"parity beyond group", fecKindParity, 255},
	} {
		t.Run(tc.name, func(t *testing.T) {
			send, recv := newLossyPair(nil)
			rx, err := newFECCarrier(recv, FECConfig{Data: 4, Parity: 2})
			if err != nil {
				t.Fatal(err)
			}
			packet := func(kind, index byte, payload string) []byte {
				wire := make([]byte, fecHeaderLen+fecLenPrefix+len(payload))
				putFECHeader(wire, kind, 1, index)
				binary.BigEndian.PutUint16(wire[fecHeaderLen:], uint16(len(payload)))
				copy(wire[fecHeaderLen+fecLenPrefix:], payload)
				return wire
			}
			if _, err := send.WriteTo(packet(tc.kind, tc.index, "malformed"), recv.addr); err != nil {
				t.Fatal(err)
			}
			if _, err := send.WriteTo(packet(fecKindData, 0, "healthy"), recv.addr); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 256)
			n, _, err := rx.ReadFrom(buf)
			if err != nil || !bytes.Equal(buf[:n], []byte("healthy")) {
				t.Fatalf("next valid packet: got %q, err=%v", buf[:n], err)
			}
		})
	}
}

func TestMalformedParityCannotPoisonFECRepair(t *testing.T) {
	for index := byte(0); index < 4; index++ {
		t.Run(fmt.Sprintf("data slot %d", index), func(t *testing.T) {
			send, recv := newLossyPair(nil)
			tx, err := newFECCarrier(send, FECConfig{Data: 4, Parity: 2})
			if err != nil {
				t.Fatal(err)
			}
			rx, err := newFECCarrier(recv, FECConfig{Data: 4, Parity: 2})
			if err != nil {
				t.Fatal(err)
			}
			// A forged parity header names a data slot in the next group. Its
			// bytes must not be used when real parity repairs a lost packet.
			bad := make([]byte, fecHeaderLen+16)
			putFECHeader(bad, fecKindParity, 0, index)
			if _, err := send.WriteTo(bad, recv.addr); err != nil {
				t.Fatal(err)
			}
			send.drop = func(n int) bool { return n == 1 } // lose the first genuine data packet
			var want [][]byte
			for i := range 4 {
				msg := []byte(fmt.Sprintf("packet-%d", i))
				want = append(want, msg)
				if _, err := tx.WriteTo(msg, recv.addr); err != nil {
					t.Fatal(err)
				}
			}
			got := drain(t, rx)
			if len(got) != len(want) {
				t.Fatalf("repair delivered %d packets, want %d: %q", len(got), len(want), got)
			}
			for _, msg := range want {
				found := false
				for _, packet := range got {
					found = found || bytes.Equal(packet, msg)
				}
				if !found {
					t.Errorf("repair lost or corrupted %q: %q", msg, got)
				}
			}
		})
	}
}
