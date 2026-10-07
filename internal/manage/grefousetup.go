package manage

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"

	"github.com/backpack/backpack/internal/tui"
)

// GRE-FOU keeps geography independent of connection direction. Its peer link
// therefore carries an explicit mode; the classic spoof wizards stay manual.
func showGREFOUPeerLink(cfg l3Spec) {
	var c config.Config
	if _, err := toml.Decode(cfg.render(), &c); err != nil {
		tui.Error(err.Error())
		return
	}
	host := ""
	if cfg.Mode == "listen" {
		detected := PublicIPv4()
		if outerIPv6(cfg.Addr) {
			detected = PublicIPv6()
		}
		if detected == "-" {
			detected = ""
		}
		host = tui.PromptDefault("This server's reachable IP or domain (for the peer link)", detected)
	}
	link, err := shareLinkOf(cfg.Name, host, c)
	if err != nil {
		tui.Error(err.Error())
		return
	}
	tui.Info("Setup Link for the other server:")
	fmt.Println(link)
}
