package manage

import (
	"fmt"
	"net"
	"strings"

	"github.com/backpack/backpack/internal/tui"
)

// The setup link, from the operator's side.
//
// Everything below this was already built: the codec, the mirror that turns one
// side's settings into the other's, the validation, and error messages written
// for somebody holding a pasted string — "copy it again, all of it", "paste the
// setup link from the other server". All of it was reachable from nothing. The
// panel encoded a link and decoded it again in the same process as a way to
// derive a peer's config, and that was its only caller.
//
// So there was nowhere to get a link and nowhere to paste one, which is the
// whole feature missing while every part of it existed.
//
// It matters more than a convenience. A tunnel has around thirty paired
// settings, and the failure a mismatch produces is the most expensive one this
// system has: the tunnel comes up, reports itself connected, and carries
// nothing. Reading two files side by side is how that mismatch happens. A link
// is how it stops.

// showShareLink prints a tunnel's setup link for the operator to carry to the
// other server.
func showShareLink(name string) {
	tui.Clear()
	tui.Title("Setup Link")
	printShareLink(name)
	tui.PressEnter()
}

// printShareLink prints the link and what to do with it, and reports whether
// there was one to print.
func printShareLink(name string) bool {
	host := ""
	if t, ok := Find(name); ok && (t.Role == "server" || (IsDirectKind(t) && (!DialsOut(t) || strings.Contains(t.Transport, "spoof")))) {
		// A reverse server listens on all interfaces, so its config cannot tell
		// the kharej side which public address to dial. Use the detected address
		// as a starting point, and let the operator correct it for NAT, a CDN,
		// or a multi-homed server before encoding the link.
		detected := PeerSetupAddress(name, PublicIPv4())
		if detected == "-" {
			detected = ""
		}
		label := "Iran IP Or Domain (What Kharej Dials)"
		if IsDirectKind(t) {
			label = "This server’s reachable real IP or domain"
		}
		host = strings.Trim(strings.TrimSpace(tui.PromptDefault(label, detected)), "[]")
		if host == "" {
			tui.Error("A reachable real IP or domain is required to build the peer setup link.")
			return false
		}
	}
	link, err := ShareLinkFor(name, host)
	if err != nil {
		tui.Error("Could not build the link: " + err.Error())
		return false
	}
	parsed, derr := DecodeShareLink(link)
	if derr != nil {
		// A link this build made and cannot read is a bug in the codec, not in
		// the operator's tunnel, and saying so is more use than the raw error.
		tui.Error("This build produced a link it cannot read back: " + derr.Error())
		return false
	}

	fmt.Println()
	if parsed.Kind == "direct" && parsed.PeerSide() == "kharej" {
		tui.Warn("On the KHAREJ server: Setup Kharej → Direct → the same carrier →")
		tui.Warn("Setup Link, and paste this line.")
	} else {
		tui.Warn("Paste this into the OTHER server: sudo backpack → Setup from a link.")
	}
	tui.Warn("It carries everything the two ends have to agree on — the token, the")
	tui.Warn("transport, the port, and the tuning — so nothing has to be retyped.")
	fmt.Println()
	tui.Info("Meant for the " + parsed.PeerSide() + " side. Shown again any time under")
	tui.Info("Manage tunnels → this tunnel → Setup Link.")
	fmt.Println()
	fmt.Println(link)
	fmt.Println()
	tui.Warn("It contains this tunnel's token. Treat it as the secret it is: anyone")
	tui.Warn("holding it can connect to this tunnel.")
	return true
}

// setupFromLink builds this machine's end from a link made on the other one.
func setupFromLink() {
	tui.Clear()
	tui.Title("Set up from a link")
	tui.Warn("Paste the setup link from the other server. It was shown there under")
	tui.Warn("Manage tunnels → the tunnel → Setup Link.")
	fmt.Println()

	raw := strings.TrimSpace(tui.Prompt("Link: "))
	if raw == "" {
		return
	}

	link, err := DecodeShareLink(raw)
	if err != nil {
		// Every refusal from the decoder is written for a person and says which
		// of them it was, so it is shown as it is rather than wrapped.
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}

	form := MirrorForPeer(link)
	if needsPeerServerAddress(form) {
		tui.Warn("This setup link is missing the other server’s reachable address.")
		label := "Iran IP Or Domain (What Kharej Dials): "
		if form.Kind == "direct" {
			label = "Other server’s reachable real address: "
		}
		host := strings.Trim(strings.TrimSpace(tui.Prompt(label)), "[]")
		var addressErr error
		form, addressErr = withPeerServerAddress(form, host)
		if addressErr != nil {
			tui.Error(addressErr.Error())
			tui.PressEnter()
			return
		}
	}
	if err := completePeerFormPorts(&form); err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	fmt.Println()
	tui.Info("This will build the " + form.Side + " end of a " + form.Kind + " tunnel.")
	tui.Info("Name       : " + form.Name)
	tui.Info("Tunnel port: " + form.TunnelPort)
	if form.Transport != "" {
		tui.Info("Transport  : " + transportLabel(form.Transport))
	}
	if form.ServerAddr != "" {
		tui.Info("Other end  : " + form.ServerAddr)
	}
	fmt.Println()

	// Iran exposes the forwarded ports; incomplete links are filled above.
	if form.Ports != "" {
		tui.Info("Ports      : " + form.Ports)
		fmt.Println()
	}

	if !tui.Confirm("Create this tunnel", true) {
		return
	}

	service, active, err := applyPeerForm(form)
	if err != nil {
		tui.Error("Failed: " + err.Error())
		tui.PressEnter()
		return
	}
	if active {
		tui.Success("Created and running: " + service)
	} else {
		tui.Warn("Created, but " + service + " is not running yet — check its log.")
	}
	tui.PressEnter()
}

// needsPeerServerAddress identifies incomplete links that need a reachable
// remote address before the peer can dial or send spoofed packets.
func needsPeerServerAddress(f PeerForm) bool {
	if f.Kind == "reverse" {
		return strings.EqualFold(f.Side, "kharej") && strings.TrimSpace(f.ServerAddr) == ""
	}
	return f.Kind == "direct" && ((((f.Mode == "" && f.Side == "iran") || f.Mode == "dial") && strings.TrimSpace(f.ServerAddr) == "") || (f.Carrier == "spoof" && strings.TrimSpace(f.SpoofPeerIP) == ""))
}

// withPeerServerAddress completes legacy or incomplete links before the
// create step, where the missing value would otherwise surface as a generic
// validation error after the operator has already confirmed the tunnel.
func withPeerServerAddress(f PeerForm, host string) (PeerForm, error) {
	if !needsPeerServerAddress(f) {
		return f, nil
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return f, fmt.Errorf("the other server’s reachable address is required")
	}
	if f.Kind == "reverse" || (f.Mode == "" && f.Side == "iran") || f.Mode == "dial" {
		f.ServerAddr = host
	}
	if f.Kind == "direct" && f.Carrier == "spoof" {
		if net.ParseIP(host).To4() == nil {
			return f, fmt.Errorf("spoof requires the peer’s real IPv4 address")
		}
		f.SpoofPeerIP = host
	}
	return f, nil
}

// applyPeerForm creates whichever kind of tunnel the form describes.
func applyPeerForm(f PeerForm) (service string, active bool, err error) {
	if f.Kind == "direct" {
		d := f.ToNewDirectTunnel()
		return CreateDirectTunnel(d)
	}
	t := f.ToNewTunnel()
	return CreateTunnel(t)
}

// SetupFromLink is the menu's entry point. Exported because internal/menu owns
// the main menu and this package owns everything it dispatches to.
func SetupFromLink() { setupFromLink() }

func completePeerFormPorts(f *PeerForm) error {
	if f.Side != "iran" {
		return nil
	}
	if strings.TrimSpace(f.Ports) == "" {
		f.Ports = strings.TrimSpace(tui.Prompt("Ports to expose on Iran (e.g. 443=127.0.0.1:2096): "))
	}
	if strings.TrimSpace(f.Ports) == "" {
		return fmt.Errorf("at least one forwarded port is required on Iran")
	}
	return validatePortSpecs(parsePorts(f.Ports))
}
