# Set up a direct XDI (ICMP) tunnel

The current **Direct → XDI** wizard builds an L3 tunnel: private IP packets are
wrapped in GRE, protected by a Noise session, and carried in ICMP echo packets.
The `Wrapping: GRE + Noise` label describes the inner framing and encryption;
XDI describes the outer carrier. This setup does not use the legacy KCP
control-channel and connection-pool transport.

Both ends need Linux and root for the TUN interface and raw ICMP sockets. The
route must pass ICMP echo traffic; successful ordinary ping is a useful first
check, but does not guarantee that larger or sustained tunnel traffic passes.

## IPv6

To use an IPv6 route, enter kharej's real IPv6 address on Iran and import the
resulting Setup Link on kharej. The link selects the ICMPv6 listener. For
manual kharej setup, enable **Listen over IPv6**; in the panel select **Outer
IP family → IPv6**. If setting kharej up first, give its reachable IPv6 address
when generating the link. Upgrade both ends to a build with IPv6 XDI support.
Keep the private tunnel addresses as they are. See [IPv6 setup details](../docs/l3-direct-tunnel.md#outer-ipv6-endpoints).

## Setup

1. On Iran run `sudo backpack` → **Setup Iran → Direct → XDI**.
2. Enter the kharej IP/domain, tunnel number and at least one forwarded port.
   ICMP has no TCP/UDP port: the tunnel number is a shared setup value, not an
   outer TCP/UDP listener to open in the firewall.
3. Keep or edit the private-address defaults. The local address includes its
   prefix, such as `10.10.0.1/30`; the peer address is bare, such as `10.10.0.2`.
4. Finish the name, token and tuning prompts, then create the tunnel.
5. On kharej use **Setup from a link** and paste Iran's generated link. It carries
   the chosen pair of private addresses, swapped for the other end. Manual
   setup is also available under **Setup Kharej → Direct → XDI → Manual**.

Iran initiates and exposes the forwarded ports. Kharej hosts the real services.
Allow ICMP on the path and the forwarded TCP ports on Iran; allow forwarded UDP
only when it is enabled. The link contains the token and must stay private.

Optional FEC can compensate for some packet loss at the cost of extra traffic.
Keep paired FEC settings consistent through the Setup Link or SSH pairing.
Throughput and latency depend on the route's ICMP filtering, packet size and
rate limits; they are not fixed by the carrier name.

## فارسی

XDI در منوی **Direct** یک تونل لایهٔ ۳ است: بسته‌بندی GRE و رمزگذاری Noise داخل
بسته‌های ICMP حمل می‌شوند. بنابراین نمایش **GRE + Noise** در بخش Wrapping طبیعی
است. این مسیر ستاپ از ترانسپورت قدیمی KCP استفاده نمی‌کند.

ایران شروع‌کننده است و پورت‌های کاربران روی ایران قرار می‌گیرند. از **Setup Iran
→ Direct → XDI** شروع کنید و لینک ساخته‌شده را در **Setup from a link** روی خارج
وارد کنید. آدرس محلی پیشوندی مثل `/30` دارد و آدرس طرف مقابل بدون پیشوند است.
هر دو سمت به لینوکس و root نیاز دارند و مسیر باید ترافیک ICMP را عبور بدهد.

[Full direct-tunnel walkthrough](direct-layer3.md) · [All tutorials](README.md)
