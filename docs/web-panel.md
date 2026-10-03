# Web panel

A dashboard and tunnel editor on **port 7777**, matching the CLI's look. It
shows live CPU / RAM / disk / traffic, each tunnel's state, real ping, and logs.
Open a tunnel's **Edit** screen to set a traffic quota in whole GiB (`0` =
unlimited) and choose download, upload, or both. Download/upload are counted
from the Iran user's perspective. Its **Metrics** screen shows inbound,
outbound, total, quota mode and remaining traffic, and marks an exhausted tunnel as paused. Backup,
Telegram setup and the panel password live in **Settings**.

From a tunnel's **Metrics** screen, an administrator can create a customer
status link, disable it, or rotate it to revoke the previous URL. The page shows
quota usage and forwarded port mappings, refreshes every 30 seconds, and offers
English/Persian plus light, dark, and system themes. Set the Telegram support
username there as well; exhausted quotas display a renewal message with that
contact. The link is a bearer URL, so share it only with that tunnel's customer.

Customer pages are served only at `/status/<id>`, and the read-only API only
at `/api/public/status`. Their old URLs under the administrator path return
404 without redirecting to login. On the first panel start after upgrading to
v1.8.19, a panel with saved customer links gets a new secret administrator path
once, including when those links are disabled. Find the new panel URL in
`sudo backpack` → **Web Panel**. Passwords, TOTP and customer IDs stay the same;
send customers the root `/status/<id>` URL. An explicitly root-mounted panel
or one that never had customer links keeps its address. Later starts keep the
new address. The custom panel path `status` is reserved for customer pages;
an existing panel using it also gets a replacement path to keep login reachable.

Run it on the **Iran** server, where you watch things from. To limit user
traffic on a reverse tunnel, set its quota on this entry server. The service
keeps watching its configuration after the tunnel pauses at the cap, so raising
the quota resumes it without erasing past usage.

## Getting in

The link and login code are shown in the CLI under **Web Panel** (whose settings
also cover update, panel port and password). Open the port first:

```bash
sudo ufw allow 7777
```

## Using your own certificate (certbot or any other)

If Let's Encrypt cannot verify this server from here — port 80 is taken, or
inbound traffic from its validators does not reach the machine — get the
certificate your own way and give the panel the two files.

- **Format:** PEM. The certificate file holds the certificate followed by its
  chain; the key file holds its private key. From certbot these are
  `fullchain.pem` and `privkey.pem`, for example:

  ```
  /etc/letsencrypt/live/panel.example.com/fullchain.pem
  /etc/letsencrypt/live/panel.example.com/privkey.pem
  ```

- **Where:** anywhere on the server; give full paths. Nothing is copied — the
  panel reads the files where they are.
- **Panel:** Settings → Panel access → Certificate → *HTTPS, my own
  certificate*. **CLI:** Web Panel → Certificate → *HTTPS, my own certificate*.
- Both check the pair before saving: a key that is not the certificate's, an
  expired certificate or an unreadable file is refused there and then.
- **Renewals** are picked up on the next connection, with no restart: point the
  panel at certbot's `live/` paths and let certbot renew as usual.
- If the files later disappear or become unreadable, the panel does not lock
  you out: it serves its self-signed certificate instead and says why in its
  log (`journalctl -u backpack-webui`).

For a **tunnel** (WSS / WSS Mux), the setup wizard's *Use existing
certificate/key files* does the same, and writes `tls_cert` and `tls_key` into
the server's config.

## Two-factor sign-in

The panel is root on this machine, and by default one password opens it. A
second factor is a code from an authenticator app — the ordinary kind, RFC 6238,
six digits every thirty seconds, so any app you already use works.

**Turning it on** is *Settings → Security → Two-factor sign-in → Turn on*. The
panel shows a key to scan or type into the app, and nothing changes until you
type back the six digits it produces — an app that never got the secret cannot
lock you out. Then it shows **ten recovery codes**, once.

Repeated sign-in guesses are limited by source address: five consecutive
incorrect password or API-token attempts trigger a ten-minute lockout. A
successful password followed by a second-factor prompt allows three code
attempts before that prompt expires; sign in again to start a fresh prompt.

**Keep the recovery codes somewhere that is not this server.** Each one signs
you in once, in the same box as the code, and they are the way back if the phone
is gone. Fresh ones can be issued at any time from the same screen, which
retires the old set.

**If the phone and the codes are both gone**, the way back is the machine
itself: CLI → **Web Panel** → **Two-factor sign-in** → turn it off. That asks
for no password on purpose. Anyone who can run it is already root on the server
and can read the file the secret is in, so a prompt would protect nothing and
would strand an operator who had also forgotten the password.

**What it does not protect.** API tokens are a separate credential and are not
affected — a token is for things that are not browsers, and a second factor has
nothing to prompt. Sessions already signed in stay signed in; sign them out from
the same Security pane if that matters.

## See also

- [Managed servers (nodes)](managed-servers.md) — registering a foreign server
  with this panel and building both ends of a tunnel from one screen. New panel
  only.

---

<div dir="rtl">

## خلاصهٔ فارسی

یک داشبورد و ویرایشگر تونل روی **پورت ۷۷۷۷** با ظاهری هماهنگ با CLI: پردازنده،
حافظه، دیسک و ترافیک زنده، وضعیت هر تونل، پینگ واقعی و لاگ‌ها. پشتیبان‌گیری،
تنظیمات تلگرام و رمز پنل در بخش **Settings** است.

روی سرور **ایران** اجرایش کن؛ در صفحهٔ **Edit** تونل می‌توانی سقف حجم را به
گیگابایت تنظیم کنی و در **Metrics** ورودی، خروجی، مصرف کل، سقف و باقیمانده را
ببینی. وقتی حجم تمام شود، تونل متوقف می‌شود؛ افزایش سقف آن را با حفظ مصرف قبلی
راه می‌اندازد.

در صفحهٔ **Metrics** هر تونل، مدیر می‌تواند لینک وضعیت مشتری را بسازد، غیرفعال
کند یا با عوض‌کردنش لینک قبلی را باطل کند. صفحهٔ مشتری مصرف و باقیماندهٔ حجم،
و پورت‌های فورواردشده را نشان می‌دهد و فارسی/انگلیسی و تم روشن/تیره/سیستم دارد.
نام کاربری پشتیبانی تلگرام را هم همان‌جا تنظیم کن؛ وقتی حجم تمام شود پیام تمدید
همراه لینک پشتیبانی نمایش داده می‌شود. این لینک مثل رمز دسترسی است؛ فقط برای همان
مشتری بفرست.

**ورود دو مرحله‌ای:** پنل روی این سرور root است و به‌صورت پیش‌فرض فقط یک رمز
جلوی آن است. از `Settings → Security → Two-factor sign-in` می‌توانی کد یک‌بارمصرف
اپلیکیشن authenticator را روشن کنی؛ تا وقتی شش رقمی که اپ نشان می‌دهد را برنگردانی
چیزی فعال نمی‌شود. بعدش **ده کد بازیابی** یک‌بار نشان داده می‌شود — آن‌ها را جایی
بیرون از همین سرور نگه دار. اگر هم گوشی و هم کدها را از دست دادی، از خود سرور:
`CLI → Web Panel → Two-factor sign-in` و خاموشش کن؛ آنجا رمز نمی‌پرسد، چون هر کسی
که بتواند آن را اجرا کند همین حالا root است.

برای مهار حدس‌زدن رمز، پنج تلاش ناموفق پیاپی از یک نشانی IP، ورود را ۱۰ دقیقه
قفل می‌کند. پس از رمز درست، برای واردکردن کد مرحلهٔ دوم سه فرصت در یک درخواست
سه‌دقیقه‌ای وجود دارد.

**ورود:** لینک و کد ورود در CLI زیر گزینهٔ **Web Panel** نشان داده می‌شود (پورت،
رمز و گواهی پنل هم همان‌جا تنظیم می‌شود). اول پورت را باز کن:
`sudo ufw allow 7777`.

</div>

**Every screen it has** — what each one shows, its address, and the CLI entry
that does the same job — is in
[the web panel, screen by screen](web-panel-screens.md).

---
[← Back to the docs index](README.md)

---

*Last verified against Backpack v1.8.20.*
