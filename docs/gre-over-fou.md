# GRE over FOU

Backpack's `gre-fou` carrier sends GRE directly inside UDP (Foo-over-UDP),
using the servers' real addresses. It keeps the layer-3 engine's authenticated
Noise session, replay protection, automatic MTU discovery, TCP segment clamp,
forwarded TCP/UDP ports, traffic accounting and quotas.

## Setup

Update both servers to a build containing this carrier. On the Iran server,
run `sudo backpack`, choose **Setup Iran → GRE over FOU**, then choose:

- **Direct:** Iran initiates; kharej listens on the chosen UDP port.
- **Reverse:** kharej initiates; Iran listens on the chosen UDP port.

Enter the peer address on the initiating side, use the same token and UDP
port at both ends, and enter the forwarded ports on Iran. The listener must
allow that UDP port through its firewall. The wizard prints a Setup Link for
the other server; paste it into **Setup from a link**. It carries the correct
opposite connection mode and swaps the private addresses while preserving
their network prefix. A link from kharej asks for Iran's forwarded ports.

In the web panel, choose **Direct → GRE over FOU**, then select direct or
reverse in **GRE over FOU connection direction**. This carrier uses the
layer-3 setup form in both directions. Managed-server setup mirrors the mode
to the other server automatically.

Editing the MTU or forwarded ports preserves the connection direction and
geographic side. Existing tunnels are not converted. Keep their configurations
until the new tunnel has been tested.

## Configuration

For reverse initiation, the Iran configuration is:

```toml
[l3]
side = "iran"
mode = "listen"
carrier = "gre-fou"
addr = "0.0.0.0:9000"
token = "USE_THE_SAME_LONG_SECRET_ON_BOTH_SERVERS"
iface = "bp0"
local_ip = "10.10.0.1/30"
peer_ip = "10.10.0.2"
mtu = 1400
ports = ["2082", "2095"]
```

The kharej configuration has `side = "kharej"`, `mode = "dial"`,
`addr = "IRAN_REAL_IP:9000"`, `local_ip = "10.10.0.2/30"` and
`peer_ip = "10.10.0.1"`, with no forwarded ports. For direct initiation,
Iran uses `mode = "dial"` and the kharej address; kharej uses `mode = "listen"`
and a local bind address. The geographic sides and private addresses stay the
same. Omitting `side` retains the older convention: dial means Iran and listen
means kharej.

## Wire format and compatibility

This is an encrypted Backpack-to-Backpack tunnel, implemented in userspace.
It does not require `ip fou add`, a kernel FOU module, or a raw IP socket.
Creating the TUN interface still requires Linux and root privileges.

Each UDP payload starts with a four-byte RFC 2784 GRE header: flags/version
zero and local experimental EtherType `0x88b5`. Its payload is Backpack's
session datagram (or the optional FEC wrapper), not a plaintext IP packet.
The inner IP packet and inner GRE key are authenticated by the Noise session.
Both endpoints must run this carrier; a plaintext Linux GRE-over-FOU interface
is not compatible with its encrypted payload.

The outer overhead is 32 bytes on IPv4 and 52 on IPv6, plus the session,
inner GRE and optional FEC overhead accounted for by the engine. Keep the
default MTU initially and let automatic discovery adjust it.

GRE over FOU does not forge an address, provide TCP retransmission, or disguise
traffic as ICMP. It requires a route that carries UDP. A route that drops UDP
will still need a different carrier.

## خلاصهٔ فارسی

<div dir="rtl">

این carrier بسته‌های GRE را داخل UDP و با آی‌پی واقعی حمل می‌کند. رمزگذاری
Noise، محدودیت ترافیک و ثبت مصرف حفظ می‌شوند. در «Setup Iran → GRE over FOU»
جهت را انتخاب کنید: دایرکت یعنی ایران شروع‌کننده است؛ ریورس یعنی خارج
شروع‌کننده است. پورت‌های کاربران در هر دو حالت روی ایران می‌مانند. لینک ستاپ
جهت طرف مقابل و آدرس‌های خصوصی را خودکار منتقل می‌کند. هر دو سرور باید
بک‌پک سازگار داشته باشند؛ این قالب رمزگذاری‌شده با GRE سادهٔ کرنل لینوکس
سازگار نیست. مسیر باید UDP را عبور بدهد و پورت UDP سمت شنونده باز باشد.

</div>

Last verified against Backpack v1.8.22.
