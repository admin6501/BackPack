package socks

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestProxyDestinationsRefuseLocalAndMetadataAddresses(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "localhost", "::1", "169.254.169.254", "ff02::1", "fd00:ec2::254"} {
		if _, err := DialTarget("tcp", net.JoinHostPort(host, "80"), time.Second); !errors.Is(err, ErrRefusedTarget) {
			t.Errorf("%s: got %v, want refused before connecting", host, err)
		}
	}
	for _, host := range []string{"8.8.8.8", "10.1.2.3", "2001:4860:4860::8888"} {
		if !PublicOrPrivate(net.ParseIP(host)) {
			t.Errorf("%s is a usable destination", host)
		}
	}
	r := &udpRelay{flows: map[string]*udpFlow{}}
	r.forward("169.254.169.254:53", []byte("x"), nil)
	if len(r.flows) != 0 {
		t.Fatal("UDP ASSOCIATE opened an outbound flow to metadata")
	}
}
