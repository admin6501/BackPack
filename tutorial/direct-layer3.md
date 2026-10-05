# Set up a direct full-IP tunnel

Use this when Iran must **dial kharej**, rather than waiting for kharej to dial
Iran. The wizard creates a private `bpN` interface on both Linux servers and
wraps GRE inside Backpack's encrypted Noise session. It can also expose ports
on Iran. This page covers the ordinary carrier wizard; **IP Spoofing** and
**SNI** keep a separate classic setup flow.

## Prepare

- Both machines need Linux, root and `/dev/net/tun`.
- Choose a tunnel port reachable **on kharej**. `9000` is the wizard default;
  allow it in kharej's host and provider firewall for the carrier you choose.
- Know kharej's real public IP or domain. A forged IP is not the answer to
  the normal wizard's reachability question.

## Start on Iran

Run `sudo backpack` → **Setup Iran → Direct** and select a carrier such as
**UDP**, **Quic**, **PCK** or **xDi**. There is no separate "Full IP tunnel"
question. The questions are:

| Prompt | Example / action |
|---|---|
| Kharej IP Or Domain | kharej's real, reachable address |
| Tunnel Port | `9000` |
| Forwarded Ports (Required) | `2095` to expose one user port on Iran; at least one valid mapping is required |
| This Server's Tunnel Address | accept the suggested free address, such as `10.10.0.1/30` |
| The Kharej Server's Tunnel Address | accept the paired address without a prefix, such as `10.10.0.2` |
| Tunnel Name | choose a distinct name |
| Security Token | accept the generated token and keep the setup link private |
| Carry UDP As Well As TCP On Those Ports | `y` only if those forwarded services need UDP |
| Turn On Error Correction (FEC) | normally `n`; use it on a measured lossy route and configure both ends alike |
| Fine-Tune The Advanced Settings | normally `n` |

Read the summary and the **setup link** before confirming. The wizard selects
an unused interface and suggests an unused private subnet; the manual prompts
let you change the two addresses. The setup link carries that pair to kharej. After creating this end, you can answer **yes** to **Set up the other
server over SSH now**; see [Set up both ends](set-up-both-ends.md). That route
installs Backpack on kharej if needed and mirrors the settings.

If you set up kharej yourself, run `sudo backpack` there → **Setup Kharej →
Direct**, select the carrier and choose **Setup Link**. Paste
the complete link printed on Iran. It fills in the peer port, token and
opposite tunnel addresses. A mismatched carrier in the initial selection is
corrected to the one carried by the link. Keep this link private: it contains
the shared token.

For **Spoof** choose the [IP Spoofing walkthrough](ip-spoofing.md) instead; that
carrier asks for both real peer IPv4 addresses and forged sources separately.

## Check the result

First check **Manage → Manage Tunnels** and **Manage → Tunnel Metrics**. Then
send traffic through one exposed port. If you left forwarded ports blank, test
the private link from Iran with `ping 10.10.0.2`, using the actual peer address
from your tunnel summary if it differs. A service shown as running only proves
the process started; it does not prove the application on kharej answers.

If the tunnel cannot establish, check the carrier and token, the listener on
kharej, and its inbound firewall. If the peer connects but a forwarded port
fails, check the service listening on kharej's mapped address and port. For
MTU or FEC adjustments, see [the direct reference](../docs/l3-direct-tunnel.md).

---

<div dir="rtl">

## خلاصهٔ فارسی

در دایرکت، **ایران به خارج وصل می‌شود**؛ پورت ورودی خود تونل باید روی خارج باز
باشد. از `sudo backpack → Setup Iran → Direct` شروع کن، حامل را انتخاب کن، آی‌پی
واقعی خارج و پورت تونل را بده، پورت‌های کاربر را روی ایران وارد کن (یا برای یک
رابط شبکهٔ خصوصی خالی بگذار) و لینک ستاپ را نگه دار.

برای ساخت خودکار سمت خارج به **Set up the other server over SSH now** جواب
`y` بده. برای ساخت دستی، روی خارج **Setup Kharej → Direct → انتخاب حامل →
Setup Link** را بزن و لینک ایران را وارد کن. در پایان عبور واقعی ترافیک یا
پینگ آدرس خصوصی طرف مقابل را امتحان کن. حامل Spoof آموزش جدا دارد.

</div>

---
[← Back to the tutorials](README.md)
