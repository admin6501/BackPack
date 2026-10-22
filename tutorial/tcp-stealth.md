# Setting up a TCP + Stealth tunnel

A TCP tunnel wrapped in a **Noise (NNpsk0) record layer**. On the wire it is two
short bursts of what looks like random bytes, followed by an encrypted stream
that looks the same, without a TLS ClientHello or plaintext application header.
Traffic sizes, timing and the Noise handshake pattern remain visible; DPI may
still classify or block it.

**Reach for it when:** the link is DPI-filtered, plain TCP connects and then dies
under load, or the tunnel keeps being killed after a while. This is the transport
that brought a filtered Germany server back online in the field.

**Cost:** a little CPU for the encryption. That is all.

> Read [TCP](tcp.md) first. Only the differences are below.

---

## What the record layer changes

- **Encrypted content.** It removes plaintext tunnel identifiers rather than
  performing TLS camouflage. Random-looking bytes can themselves be filtered;
  success depends on the route, not just the encryption.
- **The key comes from the token.** No separate key material to distribute, and
  the pre-shared key is mixed in from the very first message.
- **A wrong token gets no reply.** A peer without the token cannot complete the
  handshake, so the server sends no application handshake response. The kernel
  still accepts TCP connections; a TCP port scan can see an open port. That is also why a token typo looks identical to a firewall
  problem.

---

## The setup

Exactly the TCP walkthrough, with:

**Select transport family → `TCP`, Select TCP transport → `TCP + Stealth`**

on **both** ends. No certificate, no domain, no extra questions. Tunnel port,
token, forwarded ports, UDP question and preset are all answered as usual.

Firewall on the Iran server: the tunnel port and the forwarded ports on `tcp`.

### Pick an unremarkable tunnel port

The port, flow duration and packet pattern remain visible. Choose a reachable
port and test it on the actual route; changing the port does not make the
transport indistinguishable from HTTPS.

---

## Verifying it is really up

```
sudo backpack  →  3. Manage  →  Status
```

If the client says it is connecting and the server shows nothing:

1. **Token.** Compare byte for byte — this is the one failure Stealth gives you
   no error message for.
2. **Transport.** Both sides must say `TCP + Stealth`; a peer on plain TCP is
   just noise to a Stealth listener.
3. **Firewall.** `ufw allow <tunnel port>/tcp` on Iran.

---

## Stealth vs WSS

They present different packet patterns; either can be blocked by a particular filter.

| | TCP + Stealth | [WSS / WSS Mux](websocket-tls.md) |
|---|---|---|
| What it looks like | encrypted Noise records | ordinary HTTPS to a website |
| Needs a certificate | no | yes (self-signed or Let's Encrypt) |
| Can sit behind a CDN | no | yes |
| A browser hitting the port sees | nothing | a normal "Welcome to nginx" page |
| A scanner sweeping for it finds | an open TCP port; no valid Noise reply without the token | a different nginx on every install |
| Overhead | lower | TLS on top of WebSocket framing |

Rule of thumb: on a bare IP where "unidentifiable traffic" is fine, use Stealth.
Where unidentifiable traffic is itself suspicious, or you want a CDN in front,
use WSS.

---

<div dir="rtl">

## خلاصهٔ فارسی

**TCP + Stealth** یک تونل TCP است که داخل یک لایهٔ رمزنگاری Noise پیچیده شده.
محتوا ظاهری شبیه بایت تصادفی دارد و TLS ClientHello ندارد؛ ولی اندازه و زمان‌بندی
بسته‌ها و الگوی handshake دیده می‌شود و DPI می‌تواند آن را مسدود کند. روی مسیر
واقعی آزمایشش کن؛ رمزنگاری تضمین تشخیص‌ناپذیری نیست.

کلید رمزنگاری از خود توکن ساخته می‌شود، پس چیز اضافه‌ای برای تنظیم نیست. اگر
توکن غلط باشد handshake برنامه کامل نمی‌شود؛ با این حال اسکن TCP می‌تواند
پورت باز را ببیند. اگر وصل نشد، اول توکن و ترنسپورت دو طرف را چک کن.

راه‌اندازی مو‌به‌مو مثل [TCP](tcp.md) است، فقط در منو **TCP + Stealth** را
انتخاب کن — روی هر دو سرور. نه دامنه لازم دارد نه گواهی.

**Stealth یا WSS؟** اگر روی آی‌پی خام کار می‌کنی و «ترافیک ناشناس» مشکلی ندارد،
Stealth؛ اگر می‌خواهی دقیقاً شبیه یک سایت HTTPS باشی یا CDN جلویش بگذاری،
[WSS](websocket-tls.md).

</div>

---
[← Back to the tutorials](README.md)
