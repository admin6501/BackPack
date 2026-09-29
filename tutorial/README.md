# Backpack tutorials

Step-by-step walkthroughs for tunnel setup and everyday operations. The
transport pages follow the reverse wizard; direct tunnels and managing both
servers have their own pages below.

**New here? Read [Before you start](before-you-start.md) once.** It covers the
two roles, the token, the ports and the firewall — the four things that account
for nearly every "the tunnel is up but nothing works" report. Every other page
assumes it.

## The direction

The transport pages below set up a **reverse** tunnel: kharej dials Iran. If
that will not come up — the provider filters inbound connections, or the tunnel
port is blocked one way — use the **direct** wizard: Iran dials kharej.

| Tutorial | Use it when | Needs |
|---|---|---|
| **[Direct tunnel](direct-layer3.md)** | the reverse tunnel will not connect; Iran dials out instead | Linux, root, a port open on kharej |
| **[Direct, stream transports](direct-tunnel.md)** | you already run a `[direct]` tunnel — the wizard no longer builds these | a port open on kharej |

## The transports

| Tutorial | Use it when | Needs |
|---|---|---|
| **[TCP](tcp.md)** | you are not sure — this is the starting point | — |
| **[TCP Mux](tcp-mux.md)** | the service opens many short connections (panels, web) | — |
| **[TCP + Stealth](tcp-stealth.md)** | the link is DPI-filtered and plain TCP dies | — |
| **[TCP + PCK](tcp-pck.md)** | TCP connects then stalls, resets or is throttled | Linux, root |
| **[UDP](udp.md)** | you are forwarding one UDP service and want no layer on top | UDP open |
| **[UDP + KCP + FEC](udp-kcp-fec.md)** | gaming, or a lossy route where TCP keeps backing off | UDP open |
| **[UDP + QUIC](udp-quic.md)** | you want to test an encrypted, self-tuning UDP carrier | UDP open |
| **[WS / WS Mux](websocket.md)** | only HTTP gets through, or you want a CDN in front | — |
| **[WSS / WSS Mux](websocket-tls.md)** | you want the tunnel to look like an ordinary HTTPS site | domain (for a real cert) |
| **[xDi (ICMP)](xdi-icmp.md)** | TCP and UDP are both filtered but ping works | Linux, root |
| **[IP Spoofing](ip-spoofing.md)** | the path blocks or counts by source address | Linux, root |

## Also worth reading

- **[Set up both ends over SSH](set-up-both-ends.md)** — create the other end
  from this server, or add a managed server in the web panel. Includes rebuilds
  and changed SSH host keys.
- **[Traffic quota and reset](traffic-quota-and-reset.md)** — set a cumulative
  allowance, read remaining traffic, and begin a new usage period safely.
- **[Adding UDP to a tunnel](udp-forwarding.md)** — Xray/3x-ui UDP, WireGuard,
  DNS and games need one switch turned on. This is the page for "TCP works, UDP
  does not".
- **[Behind a panel (X-UI / 3x-ui / Marzban)](behind-a-panel.md)** — the port
  mapping, the real client IP, and what to set inside the panel.

Reference material — what each setting *is*, rather than how to set one up —
lives in [`docs/`](../docs/). The [CLI menu reference](../docs/cli-menu.md)
documents every option in every menu, including the advanced ones these
tutorials leave at their defaults.

---

<div dir="rtl">

## خلاصهٔ فارسی

این پوشه آموزش‌های قدم‌به‌قدم راه‌اندازی است — برای هر ترنسپورت یک صفحه، و در هر
صفحه دقیقاً همان سؤال‌هایی که ویزارد می‌پرسد، به همان ترتیب، با جوابی که باید بدهی
و دلیلش.

**اگر تازه شروع کرده‌ای، اول [Before you start](before-you-start.md) را بخوان.**
دو نقش (سرور ایران / کلاینت خارج)، توکن، پورت‌ها و فایروال آنجا توضیح داده شده —
همان چهار چیزی که تقریباً همهٔ «تونل بالاست ولی کار نمی‌کند»ها از آن‌ها می‌آید.

اگر نمی‌دانی کدام ترنسپورت را بگیری، از **[TCP](tcp.md)** شروع کن؛ برای فیلترینگ
سنگین **[TCP + Stealth](tcp-stealth.md)** و برای بازی و مسیر پرافت
**[UDP + KCP + FEC](udp-kcp-fec.md)**.

اگر TCP کار می‌کند ولی UDP رد نمی‌شود (Xray/3x-ui، وایرگارد، DNS، بازی) صفحهٔ
**[Adding UDP to a tunnel](udp-forwarding.md)** را ببین — فقط یک گزینه باید روشن
شود.

برای ساخت سمت دیگر با SSH از همین سرور،
[Set up both ends](set-up-both-ends.md) را بخوان. برای تنظیم سقف حجم و شروع دورهٔ
جدید، [Traffic quota and reset](traffic-quota-and-reset.md) را ببین.

توضیح تک‌تک تنظیمات (نه آموزش راه‌اندازی) در پوشهٔ [`docs/`](../docs/) است.

</div>

---
[← Back to the main README](../README.md)
