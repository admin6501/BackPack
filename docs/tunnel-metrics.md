# Tunnel Metrics

**Manage → Tunnel Metrics** shows traffic and connection counts per tunnel, and
for **KCP** the numbers that actually explain a slow link:

- retransmits,
- lost and duplicated segments, and
- how many packets **forward error correction (FEC)** repaired.

That last one is the direct answer to "is KCP earning its overhead on my route?"

Traffic totals are counted on **every** transport and are kept across restarts,
so the numbers do not reset when a tunnel bounces (and they carry on after a
[backup restore](backup-restore.md)).

For each tunnel the CLI and panel show **incoming**, **outgoing**, **total used**,
**traffic quota** and **remaining** traffic. Total is incoming + outgoing on
this server. With no quota (`0`) the balance is unlimited; at the cap it reads
zero and the panel marks the tunnel paused. Increasing the cap resumes it
without resetting the used amount. For reverse tunnels, read the Iran entry
end when accounting for user traffic; do not sum the two machines' counters.

---

<div dir="rtl">

## خلاصهٔ فارسی

**Manage → Tunnel Metrics** ترافیک و تعداد اتصال هر تونل را نشان می‌دهد، و برای
**KCP** همان عددهایی که واقعاً توضیح می‌دهند چرا لینک کند است: ارسال‌های مجدد،
سگمنت‌های گم‌شده و تکراری، و اینکه **تصحیح خطا (FEC) چند پکت را ترمیم کرده**.

عدد آخری جواب مستقیم این سؤال است: «آیا KCP روی مسیر من ارزش سربارش را دارد؟»
اگر ترمیم‌های FEC تقریباً صفر باشد، بی‌دلیل داری پهنای باند parity می‌دهی.

آمار ترافیک روی **همهٔ** ترنسپورت‌ها شمرده می‌شود و بین ری‌استارت‌ها حفظ می‌شود،
پس با بالا و پایین شدن تونل صفر نمی‌شود (و بعد از
[بازگردانی پشتیبان](backup-restore.md) هم ادامه پیدا می‌کند).

برای هر تونل، **ورودی، خروجی، مصرف کل، سقف و باقیمانده** جدا نمایش داده می‌شود.
مصرف کل همان جمع ورودی و خروجی **این سرور** است. سقف صفر یعنی نامحدود؛ وقتی
باقیمانده به صفر برسد، پنل وضعیت توقف به‌دلیل حجم را نشان می‌دهد. افزایش سقف
تونل را با حفظ مصرف قبلی راه می‌اندازد. در ریورس، برای حجم کاربران به سرور
ورودی ایران نگاه کنید و شمارنده‌های دو سرور را با هم جمع نزنید.

</div>

---
[← Back to the docs index](README.md)

---

*Describes current main; publish these changes in the next release after v1.8.4.*
