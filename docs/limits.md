# Per-tunnel limits

**Edit → Limits** puts connection and speed caps on a single tunnel. **Manage
Tunnels → select tunnel → Traffic quota** sets a cumulative data allowance:

- **Simultaneous connections** — the most forwarded connections it will carry at
  once.
- **Throughput** — a ceiling on its total bandwidth.
- **Traffic quota** — total inbound **plus** outbound traffic in GiB. A tunnel
  pauses once the quota is reached, including after a service restart. Increase
  the quota or set it to `0` to resume. The panel's Edit form also accepts it.

Set a whole number of GiB in **Manage → Manage Tunnels → select tunnel → Traffic
quota**, or in the first tab of the panel's **Edit** screen. In **Manage → Tunnel
Metrics** or the panel's **Metrics** screen, read incoming, outgoing, total,
configured quota and remaining allowance separately. Traffic counters are
cumulative across restarts and edits. At the cap the tunnel closes forwarded
listeners; its service stays active to watch for an increased or removed cap
and starts the tunnel again after the config changes. There is no timer that
resets the monthly allowance automatically.

For a reverse tunnel, apply the quota on the Iran entry end to limit traffic
accepted from users. Each end counts its own traffic; the two ends' totals are
different observations of the same flow and must not be added together.

All are **off by default**. They are useful when several services or customers
share one link and you want to stop any one of them from taking it all.

---

<div dir="rtl">

## خلاصهٔ فارسی

از `Edit → Limits` می‌شود سقف اتصال هم‌زمان و پهنای باند گذاشت. **سهمیهٔ حجم
ترافیک** از منوی همان تونل یا پنل وب تنظیم می‌شود و مجموع ورودی و خروجی است
و با GiB سنجیده می‌شود. از منوی تونل نیز گزینهٔ `Traffic quota` در دسترس است.
هر سه پیش‌فرض **خاموش** (صفر = بی‌نهایت) هستند. با رسیدن به حجم تعیین‌شده،
تونل حتی پس از راه‌اندازی دوباره متوقف می‌ماند؛ سقف را بیشتر کنید یا صفر بگذارید.

سقف را در `Manage → Manage Tunnels → نام تونل → Traffic quota` یا تب اول صفحهٔ
**Edit** پنل وب تنظیم کنید. در **Tunnel Metrics** یا **Metrics** پنل، مقدار
ورودی، خروجی، مصرف کل، سقف و باقیمانده جداست. در ریورس برای محدودکردن کاربران،
سقف را روی سرور ورودی ایران بگذارید؛ جمع دو سرور را با هم جمع نزنید. در پایان
حجم، پورت‌های تونل بسته می‌شوند و سرویس برای دیدن تغییر کانفیگ فعال می‌ماند؛
بعد از افزایش سقف، با همان مصرف قبلی دوباره شروع می‌کند. تمدید خودکار ماهانه
و صفرشدن دوره‌ای مصرف وجود ندارد.

وقتی چند سرویس یا چند مشتری روی یک لینک مشترک‌اند به کار می‌آید تا هیچ‌کدام کل
لینک را نگیرد.

اگر سقف اتصال را زیر ۱۰ بگذاری هشدار می‌دهد — یک مرورگر به‌تنهایی بیشتر از این
اتصال باز می‌کند.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against Backpack v1.8.5.*
