# Real client IP (PROXY protocol v2)

The service behind the tunnel normally sees every connection as coming from the
tunnel itself — so a VPN panel counts all users as one device, and per-user
device limits stop working.

Turn on **Edit → Real client IP (PROXY protocol)** and Backpack prefixes each
forwarded connection with a PROXY protocol v2 header carrying the user's real IP
and port, so the backend sees each user's own address.

## Availability

The reverse-tunnel setting is available on **TCP, TCP Mux, Stealth, PCK, KCP,
QUIC, WS Mux and WSS Mux**. Plain WS/WSS and raw UDP do not expose this setting.
It applies to forwarded **TCP connections**, not forwarded UDP datagrams.
The `[l3]` and legacy `[direct]` engines have separate forwarding behavior.

## Important

**The backend must be configured to accept PROXY Protocol v2 first.** If it is
not, it reads the header as ordinary traffic and every connection breaks. It is
**off by default** for exactly this reason. Enable `proxy_protocol` on the
reverse **server/entry** side; it is not a client-side setting. Keep the backend
listener private or restricted to trusted forwarders, since it trusts the
address asserted in this header.

---

<div dir="rtl">

## خلاصهٔ فارسی

سرویسِ پشت تونل به‌طور عادی همهٔ اتصال‌ها را از طرف خودِ تونل می‌بیند — پس پنل
VPN همهٔ کاربران را یک دستگاه می‌شمارد و محدودیت تعداد کاربر از کار می‌افتد.

با روشن کردن `Edit → Real client IP` هر اتصال forward شده با یک هدر
**PROXY protocol v2** حاوی آی‌پی و پورت واقعی کاربر شروع می‌شود.

**در دسترس در تونل معکوس روی:** TCP، TCP Mux، Stealth، PCK، KCP، QUIC،
WS Mux و WSS Mux. روی WS/WSS ساده و UDP خام ارائه نمی‌شود. این هدر برای اتصال
TCP فورواردشده است، نه دیتاگرام UDP. گزینهٔ `proxy_protocol` را در سمت سرور
ورودی تنظیم کن؛ گزینهٔ سمت کلاینت نیست.

**مهم:** **اول** باید سرویس مقصد طوری تنظیم شود که PROXY Protocol v2 را قبول
کند (در X-UI/Marzban گزینهٔ inbound به نام «Accept Proxy Protocol»)، **بعد**
این را روشن کنی. اگر برعکس عمل کنی، سرویس هدر را داده می‌خواند و **همهٔ
اتصال‌ها خراب می‌شوند**. دقیقاً به همین دلیل پیش‌فرض خاموش است. دسترسی به
listener مقصد را به forwarderهای مورد اعتماد محدود کن، چون هدر، آدرس مبدأ
اعلام‌شده را به سرویس مقصد می‌دهد.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against Backpack v1.8.28.*
