# Traffic quota and resetting usage

Each server records incoming and outgoing bytes **per tunnel**. A quota is a
whole number of **GiB** for download, upload, or both; `both` is the default and
preserves the historical sum. Download/upload are from the Iran user’s perspective,
so incoming and outgoing reverse on the kharej end. `0` means unlimited.
Counters survive service restarts and updates; resetting is an explicit action.

## Give a tunnel an allowance

1. Run `sudo backpack` on the **Iran entry server** for a reverse tunnel.
   Choose **Manage → Manage Tunnels → [name] → Traffic quota**.
2. Enter the allowance in GiB, for example `4000`, and choose download, upload,
   or both. This preserves consumption already recorded. Raising a quota does
   not begin a fresh period.
3. Open **Manage → Tunnel Metrics** and read **Traffic in**, **Traffic out**,
   **Traffic total**, and **Traffic quota** for this tunnel. The displayed
   remaining amount is the allowance minus the selected traffic direction.

The web panel also accepts the quota in the tunnel's **Edit** form and displays
it under **Metrics**. At the cap, forwarded listeners pause; the service remains
available to pick up an increased quota. Setting `0` removes the cap.

## Begin a new usage period

Choose **Manage → Manage Tunnels → [name] → Reset traffic**, read the
confirmation, then answer yes. In the panel use **Reset traffic** on that
tunnel's Metrics screen. This zeros **this server's** incoming and outgoing
counters and makes the configured quota available again. It briefly stops and
starts a running tunnel; a stopped tunnel stays stopped. Historical charts
retain their earlier samples and do not treat the reset as new usage.

The other server has **independent counters**. When charging users at the Iran
entry, use that server's figures; adding both servers double counts much of the
same traffic. If you deliberately need both ends' counters to begin a new
period, reset the tunnel on each server separately. There is no automatic
monthly reset.

See [Tunnel Metrics](../docs/tunnel-metrics.md) and
[Limits](../docs/limits.md) for the exact accounting and pause behavior.

---

<div dir="rtl">

## خلاصهٔ فارسی

در **Manage → Manage Tunnels → نام تونل → Traffic quota** سقف را بر حسب GiB
وارد کن و حالت «دانلود»، «آپلود» یا «هر دو» را انتخاب کن؛ «هر دو» پیش‌فرض
است و دانلود/آپلود از دید کاربر ایران سنجیده می‌شود. `4000` یعنی چهار هزار GiB؛
`0` یعنی نامحدود. مصرف قبلی با تغییر سقف پاک نمی‌شود. باقی‌مانده را در **Manage → Tunnel Metrics** ببین.

برای شروع دورهٔ جدید، **Reset traffic** همان تونل را انتخاب و تأیید کن. فقط
شمارنده‌های **همین سرور** صفر می‌شوند؛ تونل فعال لحظه‌ای دوباره وصل می‌شود.
برای حساب حجم کاربران در تونل ریورس، آمار سرور ورودی ایران را بخوان و اعداد
ایران و خارج را با هم جمع نکن. ریست ماهانهٔ خودکار وجود ندارد.

</div>

---
[← Back to the tutorials](README.md)
