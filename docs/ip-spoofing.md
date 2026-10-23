# IP Spoofing

> **IP Spoofing is a direct-tunnel carrier now, not a reverse transport.**
> `transport = "spoof"` is refused at startup, and the wizard no longer offers
> it under Setup → Experimental. Build it as a **direct tunnel** instead —
> `sudo backpack` → Setup Iran / Setup Kharej → **Direct**, then choose **Spoof**
> as the carrier — and the same forged packets carry the same forwarded ports.
>
> The reason is not tidiness. A reverse tunnel is a control channel plus a pool
> of connections, each one its own session, and a forged-source packet carries
> nothing a receiver can tell those sessions apart by: every one of them arrives
> at the same address. The session layer keys on that address, so they collapsed
> onto one and each new session closed the one before it — the tunnel reported
> itself connected and carried nothing. A direct tunnel has one session, which
> is the shape this carrier can serve.

The reference page for the direct `spoof` carrier: what it is, and what every single
setting does. For a step-by-step first setup, use
**[the IP Spoofing tutorial](../tutorial/ip-spoofing.md)** instead.

```
Setup Iran / Setup Kharej → Direct → IP Spoofing
```

Linux only. Needs root (`CAP_NET_RAW`). **Both ends must be on it.**

---

## What it is

The carrier writes its own IP packets and replaces the source address in the
on-wire header with one you choose. Routing still uses the **real** peer — the
server's bind address, the client's remote address — so the packet actually
arrives; only the header says something else.

Above the packet layer it is the [direct tunnel](l3-direct-tunnel.md): a TUN
device on each host, whole IP packets between them, sealed in a Noise session
with replay protection. The forged packets are what carries that.

It is for a path that **blocks, throttles or counts by source address**.

## What it is not

- Not [TCP + PCK](tcp-pck.md), which forges nothing and uses real addresses.
- Not anonymity. The packets still have to reach the peer, and the peer knows
  exactly who you are.
- Not something you can assume works. **Most of the difficulty with this
  transport is proving that a forged source survives your path at all** — see
  [the tester](#the-ip-spoofing-tester).

---

## What determines whether it works

- **Provider and hypervisor:** egress anti-spoofing (often BCP 38) may reject a
  forged source before it leaves the VPS. Neither a new shim nor Stealth disables it.
- **Both directions:** routing, cloud security groups, host firewalls and the
  selected IP protocol must permit the actual packets in each direction.
- **Host source validation:** reverse-path filtering (`rp_filter`) can discard
  forged sources. Linux uses the maximum of `all` and the receiving interface's
  setting; inspect both. Use an appropriate loose/disabled policy for the
  intended interface only after checking the host's routing requirements.
- **Privileges and local rules:** Linux raw sockets need `CAP_NET_RAW` (normally
  root); TCP socket receive needs the RST guard. Optional XDP has extra kernel
  and capability requirements described below.
- **Pairing:** real peer addresses, token, direction profiles, wire-changing
  flags and source pins must agree. A single source pin cannot accept a pool.
- **Packet size and load:** path MTU, fragment filtering, loss, rate limits,
  socket buffers and CPU affect speed. FEC can repair some loss; it cannot make
  a blocked protocol or source pass.

## The paired settings

Everything in a spoof setup is paired, and the pairing is what goes wrong. This
table is the whole compatibility contract:

| Setting | Must match the peer? |
|---|---|
| Packet profile / per-direction profiles | **yes** — mismatched profiles carry nothing |
| ICMP reply mode | **yes** (`icmp` and `icmpv6` profiles) |
| Padding, fake TLS | **yes** — they change the wire |
| Forged source ↔ the peer's "expected forged source" | **yes**, if you pin it |
| The other end's real IPv4 | required on **both ends** by the direct Spoof wizard |
| TTL jitter, DSCP, port shuffle, interface, XDP interface, socket buffer, MTU | no — local only |
| Error correction (`fec_data` / `fec_parity`) | **yes** — a receiver expecting a different scheme rebuilds nothing |

The last row is a **carrier** setting, not a spoof one: it lives on the tunnel
and works over every carrier. IP spoofing does not add a way to spread over
several sockets (`paths`) — that carrier already varies its source per packet,
so spreading it would add nothing. Both are documented in
[the direct tunnel reference](l3-direct-tunnel.md#error-correction).

To toggle FEC after setup, use **Edit → Error correction (FEC)**: choice **6 on
Iran**, **4 on kharej**. This is separate from the IP Spoofing submenu. Enabling
from off uses 10 data + 3 spare packets; an enabled custom pair is preserved.
Set the same pair on both ends. Saving restarts this machine's tunnel.

برای تغییر تصحیح خطا، در منوی اصلی Edit گزینهٔ **Error correction (FEC)** را بزن:
**۶ در ایران و ۴ در خارج**. این گزینه جدا از زیرمنوی IP Spoofing است؛ تنظیم دو
سرور باید یکسان باشد.

---

## Every setting

Reached from **Manage → Edit → IP Spoofing** (or the panel's spoof drawer). The
config key is given for each, for `/etc/backpack/<name>.toml`.

### Packet profile — `spoof_profile`

What the forged packets are dressed as.

Every profile below uses an **outer IPv4 header** (20 bytes before options).
The shim sizes exclude Noise, framing, optional FEC and padding overhead.

| Value | IPv4 protocol | Shim bytes | What must cross the path |
|---|:--:|:--:|---|
| **`udp`** (default) | 17 | 8 | UDP datagrams, with the chosen source accepted. Start here when UDP is allowed. |
| `tcp` | 6 | 20 | Raw SYN segments with payload. This is **not a TCP connection**; stateful filters and SYN proxies may reject it. The socket receive path manages an iptables RST guard. |
| `icmp` | 1 | 8 | Echo messages with payload. Ping success alone does not prove this packet size, rate or forged source will pass. |
| `icmpv6` | 58 | 8 | An echo-shaped shim **inside IPv4**, not real IPv6. Filters may reject this unusual protocol combination. |
| `proto58` | 58 | 0 | Bare encrypted payload under protocol 58. It has no echo header and is not wire-compatible with `icmpv6`. |
| `ipip` | 4 | 0 | Backpack payload under the IP-in-IP protocol number. It is not a plain inner IP packet for a kernel IPIP peer. |
| `gre` | 47 | 4 | Backpack payload after a minimal GRE shim. It requires a Backpack peer; it is separate from the `gre-fou` carrier. |

Only `proto58`, `ipip` and `gre` lack a port **or echo identifier** for kernel
flow filtering. The two echo profiles carry a token-derived identifier. For the
three bare profiles, a single [`spoof_peer_src_ip`](#describing-the-other-end)
can narrow receive filtering; leave it empty if the peer rotates several
sources, or the other sources will be dropped. Noise authenticates the session.

UDP/TCP destination ports and echo identifiers are derived from the tunnel
**token**. Do not assume the port displayed in the L3 address is the actual
spoof wire port; firewall rules must allow the actual carrier packets.

**Both ends must use the same profile pair and token.** A profile changes packet
format, not encryption strength, reliability or the provider's anti-spoof policy.
There is no profile guaranteed to work or be fastest on every route.

### Per-direction profiles — `spoof_uplink`, `spoof_downlink`

A different profile each way, for a path whose filtering is not symmetric — ICMP
survives one direction while UDP survives the other, say. **Uplink is the
dialling client → listening server; downlink is the reverse.** In the default
Direct setup Iran dials kharej, so uplink is Iran → kharej. Use the actual
initiation roles if you change them. Empty falls back to `spoof_profile`.

Start with a symmetric pair. If measurements justify different profiles, both
ends must configure the same uplink/downlink pair, not swap the two values.

### Forged source IP(s) — `spoof_src_ip`, `spoof_src_pool`

The address stamped on the packets this machine sends.

- **Empty forges nothing.** The tunnel runs on the machine's real address and
  still works. This is the correct first-run answer.
- **One address** goes in `spoof_src_ip`.
- **Several, comma separated**, become `spoof_src_pool`: the carrier rotates
  through the addresses **for each outgoing packet**, not per session.
  `spoof_src_ip`, if set, is also a member. Every member must pass the path;
  one failing member causes recurring loss. A peer pinned to a single
  `spoof_peer_src_ip` rejects packets from the other members.

Only use sources verified on the actual route. The
[tester](#the-ip-spoofing-tester) checks UDP; verify the selected tunnel profile
too. An address that does not pass can leave the service running while no
application traffic crosses.

### The other end's real IPv4 — `spoof_peer_ip`

**Both sides of the direct Spoof wizard ask for it.** On Iran, enter kharej's
real public IPv4; on kharej, enter Iran's real public IPv4. Forged packets do
not reveal the sender's real address, so replies must be routed to the address
you actually SSH into, never the forged address. The kharej setup refuses to
finish without Iran's real IPv4.

### Egress interface — `spoof_interface`

Pins the raw socket to a named device (`eth0`, …) on a multi-homed host, where the
forged source would otherwise leave by the wrong link. Empty lets the kernel
route by the real destination, which is right on a normal single-uplink VPS. Only
asked at setup when the machine actually has more than one interface.

### XDP receive fast path — `spoof_xdp_interface`

Set to a NIC name (`eth0`, …) to receive this tunnel's forged packets through an
**XDP/eBPF program in the kernel**, before the ordinary socket stack — higher
throughput and lower CPU under heavy load. Empty (the default) uses the normal
raw/UDP receive.

- **Pure Go.** The eBPF program is assembled in-process with `cilium/ebpf`; there
  is no clang, libbpf, or CGo, and the binary stays statically linked.
- **Opt-in and best-effort.** If the kernel is too old (the ring-buffer and
  `bpf_xdp_load_bytes` helpers need ~5.18+), the attach fails, or the verifier
  rejects the program, the carrier logs it and **falls back** to the
  ordinary receive — turning it on can never cost a working tunnel.
- Needs `CAP_BPF`/`CAP_NET_ADMIN` in addition to the carrier's `CAP_NET_RAW`
  (root has both).
- The `proto58`/`ipip`/`gre` profiles have no port or echo identifier to demux on, so the fast path needs
  `spoof_peer_src_ip` set to know which packets are the tunnel's; without it, it
  falls back for those profiles.
- Local only — it changes nothing on the wire, so the two ends need not agree,
  and each can use it or not.
- **Whether it attached is now in the log.** On start the tunnel says either
  `XDP receive fast path attached to eth0` or `XDP receive unavailable … <reason>`
  — so `spoof_xdp_interface` is no longer a switch with no feedback. Check the
  log to know which of the two you got before relying on it.

> XDP runs before IP reassembly, so it sees fragments rather than reassembled
> datagrams. The carrier sizes its packets under the tunnel MTU, so this only
> matters for a rare oversize packet, which is dropped and re-sent by whatever
> inside the tunnel owns it. Lower `spoof_mtu` if you see it.

### Relay mode — withdrawn

`spoof_mode` and `spoof_forward` were a shape of the reverse spoof transport: a
bare datagram relay to a local UDP socket, so that something bringing its own
reliability — WireGuard, usually — could ride over the forged-source channel
without KCP underneath.

They went with it, and nothing was lost. A direct tunnel is a private network
between the two machines: WireGuard, or anything else, is **routed over it**
rather than piped through it, which is the same traffic with one fewer moving
part and none of the "both ends must be in the same mode" that the relay needed.

## Fingerprint & evasion

**The short answer is Stealth.** The wizard asks one question — *Turn Stealth
on* — and the panel has one switch, and both set the whole group below at once:
padding, TTL and DSCP variation, a moving source port, and the fake TLS record
header where the profile carries one.

That is one answer rather than seven on purpose. Two of these settings change
what goes on the wire and so **must match at the other end**, and five do not.
Set individually, sooner or later a wire-changing one is set on a single end,
and the result is a tunnel that connects and carries nothing — the failure this
carrier is worst at explaining. As a group, "the same answer on both ends" is
one answer.

The individual keys are below, for a setup being tuned against a particular
path. **All are off by default and none is needed for a working tunnel.** Each
costs something — bandwidth, CPU, or a shape a different filter notices instead.
**Change one at a time and test.**

### Describing the other end

| Setting | Key | What it does |
|---|---|---|
| Expected forged source from the other end | `spoof_peer_src_ip` | Pins the source the peer stamps, so anything arriving with a different one is dropped **before the encryption looks at it** — a tighter, cheaper receive path. Empty accepts any source and leaves demux to the port and the encryption: safe, but noisier. Set it to the peer's only source. Leave it empty for a multi-address pool. |

### Sizing

| Setting | Key | What it does |
|---|---|---|
| Fragment sends above this many bytes | `spoof_mtu` | The largest IP packet the carrier emits before it fragments in userspace; the peer's kernel reassembles. 0 uses 1500. **Lower it on a path with a smaller MTU** so oversize packets fragment cleanly rather than being dropped. |
| Socket buffer in bytes | `spoof_sockbuf` | Sizes `SO_SNDBUF`/`SO_RCVBUF` on the carrier's sockets. **This is what lets a forged-source flow reach real bandwidth** — under a burst the kernel parks packets here instead of dropping them before the read loop drains them. 0 uses the 4 MiB carrier default; raise it on a fat, high-latency path. |

### Shape

| Setting | Key | Match peer? | What it does |
|---|---|:--:|---|
| ICMP/ICMPv6 echo: this pair asks and answers | `spoof_icmp_reply` | **yes** | Makes an ICMP tunnel read as a real ping exchange — client sends Echo Requests, server answers with Echo Replies — instead of both ends sending Requests. Purely cosmetic. Used by `icmp` and `icmpv6` only. |
| Vary the TTL from packet to packet | `spoof_ttl_jitter` | no | Varies the IP TTL across realistic OS defaults {64, 128, 255} instead of a fixed 64, blurring TTL fingerprints. |
| Vary the DSCP field | `spoof_random_dscp` | no | Varies the DSCP/ToS byte across plausible values instead of leaving it 0. |
| Prepend a fake TLS record header | `spoof_fake_tls` | **yes** | Prepends a cosmetic TLS 1.2 record header. There is no TLS handshake or HTTPS session; DPI may still identify or reject it. **TCP profile only.** |
| Randomise the source port of every packet | `spoof_shuffle_port` + `spoof_port_min` / `spoof_port_max` | no | For UDP/TCP, randomises the L4 **source** port per packet within the range, so the flow does not sit on one port. The destination port stays fixed, so demux is unaffected. Leave both at 0 for the whole ephemeral range. |
| Append random padding to each frame | `spoof_padding` + `spoof_padding_max` | **yes** | Appends 1..max random bytes to every payload (self-describing, so the receiver strips them), defeating size fingerprints. Costs exactly that much bandwidth. |

---

## The IP Spoofing Tester

```
Manage → IP Spoofing Tester
```

Tests forged **UDP** probes between two nodes. Linux only; the sender needs
root. It does not test TCP, ICMP, protocol 58, IPIP or GRE. A passing result is
specific to that source, direction, UDP port and measurement window; verify the
chosen tunnel profile with real traffic afterwards.

It is a **two-node test, and the receiver must be started first**:

**Receiver** — listens and tallies which forged sources arrive.

| Prompt | Default | Notes |
|---|---|---|
| Shared token | `backpack` | must match the sender; unrelated to the tunnel token |
| Listen UDP port | `45000` | open it in this machine's firewall |
| Probes the sender emits per IP | `5` | must match the sender |
| Capture window (seconds) | `30` | long enough for the sender to finish |
| Report IPs with loss% at or below | `20` | the pass threshold |
| Write passing IPs to file | — | optional; one address per line |

**Sender** — emits probes from a list, range or CIDR of forged sources.

| Prompt | Notes |
|---|---|
| Shared token | the same string |
| Receiver node's REAL IPv4 | not a forged one |
| Receiver's listen UDP port | `45000` |
| Probes per candidate IP | `5` |
| Candidate IPs | `1.0.0.0/24`, `8.8.4.4`, `9.9.9.0-9.9.9.255`, comma-separated, or `@file` with one spec per line |
| Send interface | optional |

The receiver prints an `SPOOF_IP / ARRIVED / LOSS%` table and the count that
passed. Feed the passing addresses to the **sender's** end of the tunnel as its
forged source(s).

**Swap the roles and run it again** to map the other direction — the two are
independent.

**Everything `0/5` means no valid probes reached this receiver during the
window.** First check the receiver is running, token/port match, real target IP,
host/cloud firewall, interface and route. Test with the sender's real source as
a baseline. If that passes but forged probes do not, upstream anti-spoofing or
source filtering is a likely cause; changing the cosmetic profile settings
cannot override a provider's source-address policy.

---

## Troubleshooting

| Symptom | Cause |
|---|---|
| Tunnel connects, carries nothing | check both directions: source filtering, profile/token/wire-setting mismatch, stale source pin, firewall and MTU |
| Nothing at all, even unforged | check real peer addresses, token/profile pair, permissions, actual carrier protocol/port and firewall |
| The Edit screen refuses to save anything | `spoof_peer_ip` is empty on a server — set it first |
| Throughput far below the link | measure loss/MTU and CPU first; check buffer ceilings, padding and loss on individual pool members |
| Large transfers fail, small ones fine | lower `spoof_mtu` to the real path MTU |
| Worked, then stopped | if a pool is in use, one of its addresses may now be dropped — retest |

---

<div dir="rtl">

## خلاصهٔ فارسی

ترنسپورت **spoof** پکت‌های IP را خودش می‌سازد و آدرس مبدأ روی هدر را با آدرسی که
تو انتخاب می‌کنی عوض می‌کند. مسیریابی همچنان با آدرس **واقعی** طرف مقابل انجام
می‌شود؛ رسیدن بسته به اجازهٔ شبکه و فایروال وابسته است. این حامل در تونل مستقیم
لایهٔ ۳ از **Noise** برای رمزنگاری استفاده می‌کند و KCP زیر آن نیست. لینوکس و
مجوز raw socket روی هر دو طرف لازم است.

**پروفایل‌ها چه می‌کنند؟** همهٔ این‌ها بستهٔ بیرونی **IPv4** دارند:

| پروفایل | رفتار و شرط عبور |
|---|---|
| `udp` | دیتاگرام UDP، شمارهٔ پروتکل ۱۷؛ نقطهٔ شروع وقتی UDP باز است. |
| `tcp` | سگمنت خام با فلگ SYN، شمارهٔ ۶؛ اتصال TCP معمولی و handshake ندارد و فایروال stateful ممکن است رد کند. |
| `icmp` | پیام Echo، شمارهٔ ۱؛ موفق بودن ping معمولی عبور بسته‌های جعلی با همین حجم و نرخ را تضمین نمی‌کند. |
| `icmpv6` | هدر Echo با شمارهٔ ۵۸ **داخل IPv4**؛ قابلیت IPv6 واقعی نیست. |
| `proto58` | شمارهٔ ۵۸ بدون هدر Echo؛ با `icmpv6` سازگار نیست. |
| `ipip` | شمارهٔ ۴ با محتوای Backpack؛ اتصال معمولی به IPIP کرنل نیست. |
| `gre` | شمارهٔ ۴۷ با هدر چهار بایتی GRE؛ با حامل مستقل `gre-fou` فرق دارد. |

هیچ‌کدام برای همهٔ سرورها بهترین نیستند. اجازهٔ دیتاسنتر برای خروج مبدأ جعلی،
فایروال هر دو سمت، فیلتر `rp_filter`، مسیر، MTU و تطبیق تنظیمات تعیین‌کننده‌اند.
Fake TLS فقط ظاهر یک هدر را اضافه می‌کند و HTTPS واقعی نمی‌سازد. عوض کردن پروفایل
محدودیت ضدجعل مبدأ در دیتاسنتر را برنمی‌دارد.

**آنچه حتماً باید در دو طرف یکی باشد:** پروفایل پکت (و پروفایل هر جهت)، حالت
ICMP reply، padding و fake TLS. **آنچه محلی است و لازم نیست یکی باشد:** TTL
jitter، DSCP، shuffle پورت، رابط شبکه، بافر سوکت و MTU.

**ویزارد Direct روی هر دو سمت آی‌پی واقعی طرف مقابل را می‌پرسد.** پاسخ باید به
همان آی‌پی واقعی برود، نه آدرس جعلی. `uplink` یعنی سمت dial به سمت listen؛ در
Direct پیش‌فرض ایران به خارج است و `downlink` جهت برعکس. هر دو سمت باید همان
جفت پروفایل را وارد کنند.

**آدرس مبدأ جعلی:** خالی یعنی هیچ جعلی انجام نمی‌شود و تونل با آدرس واقعی کار
می‌کند، به شرط باز بودن مسیر. چند آدرس با کاما یعنی چرخش **برای هر بسته**؛
اگر یکی نرسد، مرتب packet loss می‌گیری. در این حالت فیلتر تک‌آدرسی
`spoof_peer_src_ip` سمت مقابل باید خالی باشد. عبور UDP در تستر، عبور همهٔ
پروفایل‌ها را ثابت نمی‌کند؛ خود تونل را هم در هر دو جهت امتحان کن.

**Stealth:** ویزارد یک سؤال می‌پرسد — *Turn Stealth on* — و همان یک جواب کل
گروهِ ضدِ DPI را روشن می‌کند: padding، تغییر TTL و DSCP، جابه‌جایی پورت مبدأ، و
هدر جعلی TLS آنجا که پروفایل TCP باشد. یک سؤال است نه هفت‌تا، چون دوتای این‌ها
سیم را عوض می‌کنند و **باید در دو طرف یکی باشند**؛ اگر تک‌تک تنظیم شوند، دیر یا
زود یکی‌شان فقط روی یک سمت روشن می‌ماند و تونل وصل می‌شود ولی چیزی رد نمی‌کند.

**حالت relay حذف شد.** آن یک شکل از ترنسپورت spoofِ reverse بود و با خودش رفت.
چیزی از دست نرفت: تونل direct یک شبکه‌ی خصوصی بین دو ماشین است، پس وایرگارد
**روی آن route می‌شود** نه اینکه از داخلش pipe شود.

**بخش Fingerprint & evasion** همه‌اش پیش‌فرض خاموش است و هیچ‌کدام برای کارکردن
تونل لازم نیست. مهم‌ترین‌هایشان: `spoof_sockbuf` (اگر سرعت خیلی کمتر از لینک
است بالا ببر) و `spoof_mtu` (اگر انتقال‌های بزرگ خراب می‌شود پایین بیاور).

**تستر (`Manage → IP Spoofing Tester`)** دو طرفه است: اول Receiver را روی یک
سرور بالا بیاور، بعد Sender را روی سرور دیگر. جدول خروجی می‌گوید کدام آدرس‌ها
رسیده‌اند. تستر فقط **UDP** می‌فرستد. اگر همه `0/5` بود، اول اجرای Receiver،
توکن و پورت، آی‌پی واقعی مقصد، فایروال و مسیر را بررسی کن و با مبدأ واقعی تست
بگیر. اگر مبدأ واقعی می‌رسد و جعلی نمی‌رسد، فیلتر مبدأ در شبکه محتمل است.
بعد جای نقش‌ها را عوض کن تا جهت دیگر هم سنجیده شود.

</div>

---
[← Back to the docs index](README.md) · [Step-by-step tutorial →](../tutorial/ip-spoofing.md)


## A key that used to be here

`spoof_dst_ip` was offered until v1.8.2 and is gone. It described a forged
destination "written only into the cosmetic L4 shim", and there was nowhere for
it to go: an IPv4 destination lives in exactly one place — the IP header — and
that has to carry the peer's real address or the packet is not delivered. The
shim this carrier writes is UDP, TCP or ICMP, and none of those holds an
address; the only identifier in it is a port, derived from the token so both
ends agree without exchanging anything.

So the key was accepted, validated as an address, written to the file, and read
back into a field nothing downstream ever looked at. An update removes it from
any config that still has it, leaving the rest of the file exactly as it was.

If you set it, nothing changes: it was never doing anything.

---


## Peer setup and reconfiguration

When setting up the other end over SSH or with a Setup Link, the real reachable IPv4 address stays separate from the forged source. The producer's explicitly expected peer source becomes the new peer's own forged source. When that field is absent (the normal wizard case), the peer inherits the producer's own source or complete source pool. A rotating pool does not pin one source on the receiver. Iran forwarded ports are requested if the link does not contain them.

Running setup again and choosing no for Stealth clears the old Stealth settings. Choosing no for separate direction profiles removes the previous overrides; an empty forged source disables forging and selecting one source replaces an old pool.

در ستاپ طرف مقابل، اگر مبدأ جداگانه‌ای برای طرف مقابل مشخص نشده باشد، آی‌پی جعلی یا pool انتخاب‌شده به مبدأ جعلی طرف مقابل هم منتقل می‌شود. آی‌پی واقعی با آی‌پی جعلی جدا می‌ماند و پورت‌های ایران، اگر در لینک نباشند، پرسیده می‌شوند. در ستاپ مجدد، انتخاب خاموش برای Stealth یا پروفایل جداگانه تنظیم قبلی را پاک می‌کند؛ خالی گذاشتن مبدأ جعلی، جعل مبدأ را خاموش می‌کند.

*Last verified against Backpack v1.8.29.*
