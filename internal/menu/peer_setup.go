package menu

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/backpack/backpack/internal/control"
	"github.com/backpack/backpack/internal/manage"
	"github.com/backpack/backpack/internal/manage/spec"
	"github.com/backpack/backpack/internal/node"
	"github.com/backpack/backpack/internal/tui"
	"golang.org/x/sys/unix"
)

// setupWithPeer keeps the existing wizards intact. Once one has made a tunnel,
// the operator can finish its other end without opening a second terminal.
func setupWithPeer(setup func()) {
	before := map[string]bool{}
	for _, t := range manage.List() {
		before[t.Name] = true
	}
	setup()
	var created []manage.Tunnel
	for _, t := range manage.List() {
		if !before[t.Name] {
			created = append(created, t)
		}
	}
	if len(created) != 1 || !tui.Confirm("Set up the other server over SSH now", false) {
		return
	}
	setUpPeer(created[0])
}

func pairExistingTunnel() {
	tunnels := manage.List()
	if len(tunnels) == 0 {
		tui.Warn("Create this server's end of a tunnel first.")
		tui.PressEnter()
		return
	}
	opts := make([]tui.Option, 0, len(tunnels))
	for _, t := range tunnels {
		opts = append(opts, tui.Option{Title: t.Name, Desc: t.Role + " / " + t.Transport})
	}
	idx := tui.ChooseOpt("Which tunnel should be set up on the other server?", opts)
	if idx >= 0 {
		setUpPeer(tunnels[idx])
	}
}

// askSSHPassword disables terminal echo. Refuse to ask on a piped stdin: an
// echoed root password in a terminal transcript is worse than a cancelled add.
func askSSHPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	old, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return "", fmt.Errorf("a terminal is required to enter the SSH password privately: %w", err)
	}
	hidden := *old
	hidden.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &hidden); err != nil {
		return "", err
	}
	defer unix.IoctlSetTermios(fd, unix.TCSETS, old)
	password := tui.Prompt("SSH password (hidden): ")
	fmt.Println()
	if password == "" {
		return "", fmt.Errorf("an SSH password is required")
	}
	return password, nil
}

func choosePeerNode(fleet *control.Fleet) (string, error) {
	known := node.List()
	opts := []tui.Option{{Title: "Add a server", Desc: "connect over SSH and install Backpack if needed"}}
	for _, n := range known {
		opts = append(opts, tui.Option{Title: n.Name, Desc: n.Host})
	}
	idx := tui.ChooseOpt("Where is the other end?", opts)
	if idx < 0 {
		return "", nil
	}
	if idx > 0 {
		return known[idx-1].Name, nil
	}
	name := strings.TrimSpace(tui.Prompt("Server name (letters, digits, - or _): "))
	host := strings.TrimSpace(tui.Prompt("Server SSH address (IP or hostname): "))
	port := tui.PromptInt("SSH port", 22)
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("SSH port must be between 1 and 65535")
	}
	user := strings.TrimSpace(tui.PromptDefault("SSH username", "root"))
	password, err := askSSHPassword()
	if err != nil {
		return "", err
	}
	tui.Info("Connecting over SSH; Backpack will be installed there if needed...")
	joined, err := fleet.Join(name, host, port, user, password, true)
	if err != nil {
		return "", err
	}
	if joined.Installed {
		tui.Info("Backpack installed on " + name + ".")
	}
	return name, nil
}

// peerApplyRequest uses the same setup-link mirror and node operation as the
// web panel. Never retype paired transport settings or send shell commands.
func peerApplyRequest(link manage.ShareLink) (node.ApplyRequest, string, error) {
	form := manage.MirrorForPeer(link)
	if form.ServerAddr == "" && ((form.Kind == "reverse" && form.Side == "kharej") ||
		(form.Kind == "direct" && (form.Mode == "dial" || (form.Mode == "" && form.Side == "iran")))) {
		return node.ApplyRequest{}, "", fmt.Errorf("the other side needs an address for this server")
	}
	if form.Kind == "direct" {
		d := form.ToNewDirectTunnel()
		return node.ApplyRequest{Kind: "direct", Direct: &d}, form.Name, nil
	}
	if form.Kind != "reverse" {
		return node.ApplyRequest{}, "", fmt.Errorf("unsupported tunnel kind %q", form.Kind)
	}
	t := form.ToNewTunnel()
	return node.ApplyRequest{Kind: "reverse", Tunnel: &t}, form.Name, nil
}

// A listener must supply its reachable address even when it is on Iran in
// reverse L3 mode. Geography alone does not determine who initiates.
func peerSetupNeedsHost(t manage.Tunnel) bool {
	return !manage.DialsOut(t) || (manage.IsDirectKind(t) && strings.Contains(t.Transport, "spoof"))
}

func setUpPeer(t manage.Tunnel) {
	tui.Clear()
	tui.Title("Set up both ends — " + t.Name)
	if pair, ok := manage.PairFor(t.Name); ok {
		tui.Warn("This tunnel is already paired with " + pair.Node + ".")
		if !tui.Confirm("Set it up on another server instead", false) {
			return
		}
	}
	host := ""
	// A listening end's config usually says 0.0.0.0. The peer needs the
	// reachable address, which only the operator can confirm on a routed VPS.
	if peerSetupNeedsHost(t) {
		host = strings.TrimSpace(tui.PromptDefault("This server's address as the peer reaches it", manage.PeerSetupAddress(t.Name, manage.PublicIPv4())))
		if host == "" || host == "-" {
			tui.Error("A reachable address is required before setting up the other end.")
			tui.PressEnter()
			return
		}
	}
	link, err := manage.ShareLinkFor(t.Name, host)
	if err != nil {
		tui.Error("Cannot read this tunnel: " + err.Error())
		tui.PressEnter()
		return
	}
	parsed, err := manage.DecodeShareLink(link)
	if err != nil {
		tui.Error("Cannot mirror this tunnel: " + err.Error())
		tui.PressEnter()
		return
	}
	apply, peerName, err := peerApplyRequest(parsed)
	if err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	if err := completePeerPorts(&apply); err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	var fleet control.Fleet
	if err := fleet.Start(); err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	defer fleet.Stop()
	nodeName, err := choosePeerNode(&fleet)
	if err != nil {
		tui.Error("Could not connect to the other server: " + err.Error())
		tui.PressEnter()
		return
	}
	if nodeName == "" {
		return
	}
	run := fleet.Runner()
	if ok, why := run.Reachable(nodeName); !ok {
		tui.Error("The other server is unreachable: " + why)
		tui.PressEnter()
		return
	}
	var existing []node.TunnelState
	if err := run.Call(nodeName, node.OpList, nil, &existing); err != nil {
		tui.Error("Cannot check the other server's tunnels: " + err.Error())
		tui.PressEnter()
		return
	}
	for _, current := range existing {
		if strings.EqualFold(current.Name, peerName) && !tui.Confirm(
			fmt.Sprintf("%q already exists on %s. Replace its settings", peerName, nodeName), false) {
			return
		}
	}
	tui.Info(fmt.Sprintf("Local: %s  |  Other server: %s (%s)", t.Name, nodeName, peerName))
	if !tui.Confirm("Create or update the other end now", false) {
		return
	}
	var result node.ApplyResult
	if err := run.Call(nodeName, node.OpApply, apply, &result); err != nil {
		tui.Error("This end remains as it was; the other end failed: " + err.Error())
		tui.Warn("Fix that error and use Manage → Set up the other server to retry.")
		tui.PressEnter()
		return
	}
	if err := manage.NoteNodePair(t.Name, nodeName, peerName); err != nil {
		tui.Warn("Both ends were written, but pairing was not saved: " + err.Error())
	}
	if !result.Active || !manage.IsActive(t.Service) {
		tui.Warn("Both ends were configured, but a tunnel service is stopped. Check logs on both servers.")
		tui.PressEnter()
		return
	}
	tui.Success("Both tunnel services are running.")
	// Give the dialling side a moment to connect, then report the peer's own
	// answer. Running processes alone are not proof that data can pass.
	for attempt := 0; attempt < 3; attempt++ {
		var status node.TunnelState
		if err := run.Call(nodeName, node.OpStatus, node.NameRequest{Name: peerName}, &status); err != nil {
			tui.Warn("Connection check failed: " + err.Error())
			break
		}
		if status.Connected != nil && *status.Connected {
			tui.Success("The other end reports its peer connection is up.")
			break
		}
		if attempt == 2 {
			tui.Warn("The peer connection is not verified yet. Check Tunnel Metrics or run a traffic test.")
		} else {
			time.Sleep(2 * time.Second)
		}
	}
	tui.PressEnter()
}

// Only Iran exposes customer ports. Ask at the initiating end before SSH.
func completePeerPorts(apply *node.ApplyRequest) error {
	var ports *string
	if apply.Tunnel != nil && apply.Tunnel.Role == "server" {
		ports = &apply.Tunnel.Ports
	}
	if apply.Direct != nil && apply.Direct.Side == "iran" {
		ports = &apply.Direct.Ports
	}
	if ports == nil {
		return nil
	}
	if strings.TrimSpace(*ports) == "" {
		*ports = strings.TrimSpace(tui.Prompt("Ports to expose on the Iran server (e.g. 443=127.0.0.1:2096): "))
	}
	if strings.TrimSpace(*ports) == "" {
		return fmt.Errorf("the Iran server needs at least one forwarded port")
	}
	return spec.ValidatePortSpecs(spec.ParsePorts(*ports))
}
