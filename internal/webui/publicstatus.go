package webui

import (
	_ "embed"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/app"
	"github.com/backpack/backpack/internal/manage"
	"github.com/backpack/backpack/internal/metrics"
)

//go:embed assets/customer-status.html
var customerStatusHTML []byte

type publicTunnelStatus struct {
	LimitBytes    uint64   `json:"limitBytes"`
	UsedBytes     uint64   `json:"usedBytes"`
	Remaining     uint64   `json:"remainingBytes"`
	Mode          string   `json:"mode"`
	State         string   `json:"state"`
	Ports         []string `json:"ports"`
	SupportURL    string   `json:"supportUrl,omitempty"`
	UpdatedAtUnix int64    `json:"updatedAt"`
}

// handlePublicStatusPage serves the standalone customer view. The ID is a
// bearer credential, so it is never copied into a referrer or cached response.
func (s *server) handlePublicStatusPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	id := strings.TrimPrefix(r.URL.Path, "/status/")
	if r.Method != http.MethodGet || !publicIDExists(id) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(withNonce(customerStatusHTML, r))
}

func publicIDExists(id string) bool {
	if len(id) != 48 {
		return false
	}
	for _, link := range Load().PublicTunnelLinks {
		if link.Enabled && link.ID == id {
			return true
		}
	}
	return false
}

func (s *server) handlePublicStatusData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	id := r.URL.Query().Get("id")
	c := Load()
	var name string
	for tunnel, link := range c.PublicTunnelLinks {
		if link.Enabled && link.ID == id && len(id) == 48 {
			name = tunnel
			break
		}
	}
	if name == "" {
		http.NotFound(w, r)
		return
	}
	t, ok := manage.Find(name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	cfg, err := manage.LoadTunnelConfig(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	snap, err := metrics.Read(app.ConfigDir, name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		http.Error(w, "status usage data is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	mode := config.NormalizeTrafficLimitMode(cfg.TrafficLimitMode)
	used := config.TrafficQuotaUsage(mode, snap.Role, snap.BytesIn, snap.BytesOut)
	var limit uint64
	if cfg.TrafficLimitGB > 0 {
		limit = uint64(cfg.TrafficLimitGB) << 30
	}
	state := manage.TunnelHealth(t).State
	if limit > 0 && used >= limit {
		state = "exhausted"
	}
	ports := customerPorts(manage.VisiblePorts(t.Ports, cfg.Server.Token))
	support := ""
	if handle := normalizeSupportHandle(c.SupportTelegram); handle != "" {
		support = "https://t.me/" + handle
	}
	var remaining uint64
	if limit > used {
		remaining = limit - used
	}
	var updated int64
	if !snap.Taken.IsZero() {
		updated = snap.Taken.Unix()
	}
	writeJSON(w, publicTunnelStatus{
		LimitBytes: limit, UsedBytes: used, Remaining: remaining,
		Mode: mode, State: state, Ports: ports,
		SupportURL: support, UpdatedAtUnix: updated,
	})
}

// customerPorts retains the public listening port and optional backend port,
// while dropping backend addresses that could be private or customer-specific.
func customerPorts(specs []string) []string {
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		parts := strings.SplitN(spec, "=", 2)
		label := strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			target := strings.TrimSpace(parts[1])
			if _, port, err := net.SplitHostPort(target); err == nil && port != "" {
				label += " → " + port
			} else if _, port, err := net.SplitHostPort("x:" + target); err == nil && port != "" {
				label += " → " + port
			}
		}
		out = append(out, label)
	}
	return out
}

type publicLinkRequest struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
	Rotate  bool   `json:"rotate"`
}

func (s *server) handlePublicLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		name := r.URL.Query().Get("name")
		if _, ok := manage.Find(name); !ok {
			http.NotFound(w, r)
			return
		}
		link := Load().PublicTunnelLinks[name]
		out := map[string]any{"enabled": link.Enabled}
		if link.ID != "" {
			out["url"] = publicURL(r, link.ID)
		}
		writeJSON(w, out)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req publicLinkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if _, ok := manage.Find(req.Name); !ok {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}
	c := Load()
	if c.PublicTunnelLinks == nil {
		c.PublicTunnelLinks = map[string]PublicTunnelLink{}
	}
	link := c.PublicTunnelLinks[req.Name]
	if req.Rotate || link.ID == "" {
		link.ID = randomHex(24)
	}
	if req.Enabled != nil {
		link.Enabled = *req.Enabled
	}
	c.PublicTunnelLinks[req.Name] = link
	if err := Save(c); err != nil {
		http.Error(w, "could not save public link", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"enabled": link.Enabled, "url": publicURL(r, link.ID)})
}

func publicURL(r *http.Request, id string) string {
	scheme := "http"
	if secureRequest(r) {
		scheme = "https"
	}
	return scheme + "://" + r.Host + basePrefix() + "/status/" + url.PathEscape(id)
}

type publicSettingsRequest struct {
	SupportTelegram string `json:"supportTelegram"`
}

func (s *server) handlePublicSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]string{"supportTelegram": Load().SupportTelegram})
	case http.MethodPost:
		var req publicSettingsRequest
		if err := decodeJSON(w, r, &req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		handle := normalizeSupportHandle(req.SupportTelegram)
		if strings.TrimSpace(req.SupportTelegram) != "" && handle == "" {
			http.Error(w, "enter a Telegram username such as @support", http.StatusBadRequest)
			return
		}
		c := Load()
		c.SupportTelegram = handle
		if err := Save(c); err != nil {
			http.Error(w, "could not save support contact", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"supportTelegram": handle})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func normalizeSupportHandle(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "https://t.me/")
	v = strings.TrimPrefix(v, "http://t.me/")
	v = strings.TrimPrefix(v, "t.me/")
	v = strings.TrimPrefix(v, "@")
	if v == "" || len(v) < 5 || len(v) > 32 {
		return ""
	}
	for _, ch := range v {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return ""
		}
	}
	return v
}
