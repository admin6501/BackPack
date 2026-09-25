# Per-tunnel limits

**Edit → Limits** puts connection and speed caps on a single tunnel. **Manage
Tunnels → Traffic quota** also sets a cumulative data allowance:

- **Simultaneous connections** — the most forwarded connections it will carry at
  once.
- **Throughput** — a ceiling on its total bandwidth.
- **Traffic quota** — total inbound **plus** outbound traffic in GiB. A tunnel
  pauses once the quota is reached, including after a service restart. Increase
  the quota or set it to `0` to resume. The panel's Edit form also accepts it.

All are **off by default**. They are useful when several services or customers
share one link and you want to stop any one of them from taking it all.

---

<div dir="rtl">

## خلاصهٔ فارسی

از `Edit → Limits` می‌شود سه سقف روی یک تونل گذاشت: **حداکثر اتصال هم‌زمان**،
**سقف پهنای باند** و **سهمیهٔ حجم ترافیک**. سهمیهٔ حجم مجموع ورودی و خروجی است
و با GiB سنجیده می‌شود. از منوی تونل نیز گزینهٔ `Traffic quota` در دسترس است.
هر سه پیش‌فرض **خاموش** (صفر = بی‌نهایت) هستند. با رسیدن به حجم تعیین‌شده،
تونل حتی پس از راه‌اندازی دوباره متوقف می‌ماند؛ سقف را بیشتر کنید یا صفر بگذارید.

وقتی چند سرویس یا چند مشتری روی یک لینک مشترک‌اند به کار می‌آید تا هیچ‌کدام کل
لینک را نگیرد.

اگر سقف اتصال را زیر ۱۰ بگذاری هشدار می‌دهد — یک مرورگر به‌تنهایی بیشتر از این
اتصال باز می‌کند.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against Backpack v1.8.4.*
