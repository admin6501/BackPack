package manage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

func TestReversePairedSettingsFromBothProducersAcrossTransports(t *testing.T) {
	for _, tr := range []string{"tcp", "tcpmux", "udp", "kcp", "ws", "wss", "wsmux", "wssmux", "stealth", "xdi", "quic", "pck"} {
		for _, from := range []string{"iran", "kharej"} {
			t.Run(tr+"/"+from, func(t *testing.T) {
				cfg := config.Config{}
				if from == "iran" {
					cfg.Server = config.ServerConfig{BindAddr: "0.0.0.0:9000", Token: strings.Repeat("k", 64), Transport: config.TransportType(tr), Ports: []string{"443"}, SimpleAuth: true, MuxVersion: 1, KCPConfig: config.KCPConfig{DataShards: 12, ParityShards: 5, MTU: 1250}, FallbackTransports: []config.TransportType{"quic"}, FallbackDwell: 37}
				}
				if from == "kharej" {
					cfg.Client = config.ClientConfig{RemoteAddr: "203.0.113.1:9000", Token: strings.Repeat("k", 64), Transport: config.TransportType(tr), SimpleAuth: true, MuxVersion: 1, KCPConfig: config.KCPConfig{DataShards: 12, ParityShards: 5, MTU: 1250}, FallbackTransports: []config.TransportType{"quic"}, FallbackDwell: 37}
				}
				encoded, err := shareLinkOf("paired", "203.0.113.1", cfg)
				if err != nil {
					t.Fatal(err)
				}
				link, err := DecodeShareLink(encoded)
				if err != nil {
					t.Fatal(err)
				}
				f := MirrorForPeer(link)
				n := f.ToNewTunnel()
				if link.SimpleAuth != true || link.MuxVer != 1 || n.Conn == nil || !n.Conn.SimpleAuth || n.Tune == nil || n.Tune.MuxVersion != 1 {
					t.Fatal("auth/mux fields lost", link, n)
				}
				if !reflect.DeepEqual(n.FallbackTransports, []string{"quic"}) || n.FallbackDwell != 37 {
					t.Fatal("fallback chain lost", n)
				}
				if isKCP(tr) && (n.Tune.KCPDataShards != 12 || n.Tune.KCPParityShards != 5 || n.Tune.KCPMTU != 1250) {
					t.Fatal("KCP paired settings lost", n.Tune)
				}
				if from == "iran" {
					s := reverseClientFromLink(link, "203.0.113.1")
					if s.SimpleAuth != true || s.MuxVersion != 1 || s.FallbackDwell != 37 || (isKCP(tr) && (s.KCPDataShards != 12 || s.KCPParityShards != 5 || s.KCPMTU != 1250)) {
						t.Fatal("manual link wizard lost settings", s)
					}
				}
			})
		}
	}
}

func TestDirectEditorsKeepSNIDomain(t *testing.T) {
	l := config.L3Config{Mode: "dial", Carrier: "sni", SNIDomain: "allowed.example", Encap: "gre", Addr: "203.0.113.1:9000", Token: strings.Repeat("k", 64), Iface: "bp0", LocalIP: "10.10.0.1/30", PeerIP: "10.10.0.2", MTU: 1400, Ports: []string{"443"}}
	for _, s := range []l3Spec{l3SpecOf(Tunnel{Name: "missing-audit-fixture"}, l), directSpecFrom("missing-audit-fixture", l)} {
		var cfg config.Config
		if _, err := toml.Decode(s.render(), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.L3.SNIDomain != l.SNIDomain || cfg.L3.Encap != l.Encap {
			t.Fatal("unrelated edit changed SNI/encapsulation", cfg.L3)
		}
	}
}

func TestSpoofSetupAgainClearsOldDirectionAndSourcePool(t *testing.T) {
	for _, source := range []string{"", "81.28.60.3"} {
		sc := config.SpoofConfig{SpoofPeerIP: "203.0.113.1", SpoofUplink: "icmp", SpoofDownlink: "tcp", SpoofSrcIP: "81.28.60.1", SpoofSrcPool: []string{"81.28.60.1", "81.28.60.2"}}
		input := "\n\n\n" + source + "\n"
		if len(routableInterfaces()) > 1 {
			input += "\n"
		}
		input += "n\n"
		restore := tui.SetInput(strings.NewReader(input))
		askSpoofCarrier(&sc, true)
		restore()
		if sc.SpoofUplink != "" || sc.SpoofDownlink != "" || len(sc.SpoofSrcPool) != 0 || sc.SpoofSrcIP != source {
			t.Fatal("setup kept obsolete source/direction fields", sc)
		}
	}
}

func TestPckAutomaticChoiceClearsOldInterfaceAndGateway(t *testing.T) {
	s := TunnelSpec{Transport: "pck", PckInterface: "old0", PckGatewayMAC: "00:11:22:33:44:55"}
	restore := tui.SetInput(strings.NewReader("1\nn\n"))
	defer restore()
	askPck(&s)
	if s.PckInterface != "" || s.PckGatewayMAC != "" {
		t.Fatal("automatic routing retained old overrides", s)
	}
}

func TestReverseEditorsKeepLocalOverridesAcrossEveryTransport(t *testing.T) {
	for _, tr := range []string{"tcp", "tcpmux", "udp", "kcp", "ws", "wss", "wsmux", "wssmux", "stealth", "xdi", "quic", "pck"} {
		original := config.Config{}
		original.Server = config.ServerConfig{BindAddr: "0.0.0.0:9000", Token: "secret", Transport: config.TransportType(tr), SkipOptz: true, SOPinTCP: true, PPROF: true, SnifferLog: "server.log"}
		original.Client = config.ClientConfig{RemoteAddr: "203.0.113.1:9000", Token: "secret", Transport: config.TransportType(tr), RetryInterval: 17, DialTimeout: 43, SkipOptz: true, SOPinTCP: true, PPROF: true, SnifferLog: "client.log"}
		for _, s := range []TunnelSpec{serverSpecOf("fixture", original), clientSpecOf("fixture", original)} {
			var got config.Config
			if _, err := toml.Decode(s.Render(), &got); err != nil {
				t.Fatal(err)
			}
			if s.Role == "server" && (!got.Server.SkipOptz || !got.Server.SOPinTCP || !got.Server.PPROF || got.Server.SnifferLog != "server.log") {
				t.Fatal("server edit lost overrides", tr, got.Server)
			}
			if s.Role == "client" && (got.Client.RetryInterval != 17 || got.Client.DialTimeout != 43 || !got.Client.SkipOptz || !got.Client.SOPinTCP || !got.Client.PPROF || got.Client.SnifferLog != "client.log") {
				t.Fatal("client edit lost overrides", tr, got.Client)
			}
		}
	}
}

func TestAllDirectCarriersKeepPairedFieldsBothDirections(t *testing.T) {
	off := false
	for _, carrier := range []string{"udp", "pck", "quic", "sni", "xdi", "spoof"} {
		for _, from := range []string{"iran", "kharej"} {
			t.Run(carrier+"/"+from, func(t *testing.T) {
				link := ShareLink{Kind: "direct", From: from, Tr: carrier, Name: "paired", Tok: strings.Repeat("k", 64), Host: "203.0.113.1", Port: "9000", LocalIP: "10.10.0.1/30", PeerIP: "10.10.0.2", Ports: "443", MTU: 1370, FECData: 12, FECParity: 5, Paths: 2, GREKey: 7, L3MSS: 1300, AutoMTU: &off, SNI: "allowed.example", Profile: "icmp", SrcIPs: "81.28.60.1", PeerSrcIP: "8.8.4.4"}
				encoded, err := link.Encode()
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := DecodeShareLink(encoded)
				if err != nil {
					t.Fatal(err)
				}
				form := MirrorForPeer(decoded)
				s, err := form.ToNewDirectTunnel().spec()
				if err != nil {
					t.Fatal(err)
				}
				var got config.Config
				if _, err := toml.Decode(s.render(), &got); err != nil {
					t.Fatal(err)
				}
				if got.L3.Carrier != carrier || got.L3.LocalIP != "10.10.0.2/30" || got.L3.PeerIP != "10.10.0.1" || got.L3.Token != link.Tok || got.L3.MTU != 1370 || got.L3.FECData != 12 || got.L3.FECParity != 5 || got.L3.GREKey != 7 || got.L3.MSSClamp != 1300 || got.L3.AutoMTU == nil || *got.L3.AutoMTU {
					t.Fatal("paired direct field lost", got.L3)
				}
				if carrier == "sni" && got.L3.SNIDomain != "allowed.example" {
					t.Fatal("SNI lost")
				}
			})
		}
	}
}

func TestIncompletePeerLinksRequestOnlyRequiredFields(t *testing.T) {
	for _, f := range []PeerForm{{Kind: "reverse", Side: "kharej"}, {Kind: "direct", Side: "iran", Carrier: "udp"}, {Kind: "direct", Side: "kharej", Carrier: "spoof"}} {
		if !needsPeerServerAddress(f) {
			t.Fatal("missing address not detected", f)
		}
		got, err := withPeerServerAddress(f, "203.0.113.9")
		if err != nil || needsPeerServerAddress(got) {
			t.Fatal("peer address not completed", got, err)
		}
		if f.Carrier == "spoof" {
			if _, err := withPeerServerAddress(f, "peer.example"); err == nil {
				t.Fatal("spoof accepted domain instead of IPv4")
			}
		}
	}
	for _, kind := range []string{"reverse", "direct"} {
		f := PeerForm{Kind: kind, Side: "iran"}
		tui.SetInput(strings.NewReader("443, 8443\n"))
		err := completePeerFormPorts(&f)
		tui.SetInput(nil)
		if err != nil || f.Ports != "443, 8443" {
			t.Fatal("Iran ports not requested", f, err)
		}
		f.Ports = "70000"
		if err := completePeerFormPorts(&f); err == nil {
			t.Fatal("invalid port accepted")
		}
	}
}
