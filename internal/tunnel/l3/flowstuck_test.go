package l3

import (
	"context"
	"errors"
	"testing"
	"time"
)

// An unanswered direct UDP tunnel must leave its old socket so the caller can
// retry from a new source port. The fake device makes this a full Run test
// without requiring a TUN interface on the CI runner.
func TestUnansweredUDPFlowReopensInsteadOfRetryingForever(t *testing.T) {
	tun, err := New(Config{
		Mode: ModeDial, Addr: "127.0.0.1:9", Carrier: CarrierUDP,
		Token: "a-token-for-flow-recovery", LocalIP: "10.10.0.1/30",
		PeerIP: "10.10.0.2", MTU: 1400,
	}, quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	tun.flowStuckAfter = 0
	tun.openDevice = func(deviceSpec) (packetDevice, error) { return newFakeDevice(1400), nil }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- tun.Run(ctx) }()
	select {
	case err := <-done:
		if !errors.Is(err, errFlowStuck) {
			t.Fatalf("Run returned %v, want a stuck-flow restart", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("unanswered UDP flow kept retrying without reopening")
	}
}

func TestOnlyCarriersWithMovableFlowsReopen(t *testing.T) {
	for _, carrier := range []string{"", CarrierUDP, CarrierQuic, CarrierXdi, CarrierPck, CarrierSNI} {
		if !stuckFlowCarrier(carrier) {
			t.Errorf("%q should retry with a new flow", carrier)
		}
	}
	if stuckFlowCarrier(CarrierSpoof) {
		t.Fatal("spoof cannot change its forged source by reopening")
	}
}
