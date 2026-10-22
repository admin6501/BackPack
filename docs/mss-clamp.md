# TCP MSS clamp

Most tunnels never need this. Set it only when something has measured the path
and told you to.

## The fault it fixes

Some paths cannot carry a full-sized packet, and drop the oversized ones without
sending back the ICMP message that would say so. Nothing learns: the handshake
and the heartbeats are small enough to arrive, so the tunnel connects, stays
connected, and reports healthy from every angle — while every real transfer
stalls on the first full segment.

**Manage → Health Check** measures this on purpose, because nothing else does:

```
✗ Path MTU   path carries 1260 bytes but the tunnel sends segments of 1448
             → set the MSS clamp to 1208 on both ends
```

## Setting it

- **CLI** — Manage → Manage Tunnels → Edit → **TCP MSS clamp**
- **Web panel** — Edit → **Fine Tune** → *TCP MSS clamp*
- **At setup** — answer *yes* to "Fine-tune the advanced settings by hand"

For the stream `mss` setting, `0` leaves the choice to the kernel. For L3,
`mss_clamp = 0` derives an inner TCP MSS from the tunnel MTU; `-1` disables
Backpack's TUN-interface clamp. These are different settings.

**Set the same value on both ends.** Each end clamps only what *it* sends, so a
tunnel clamped on one side is still sending full-sized packets from the other —
and those are the ones being dropped.

## Notes

- The stream `mss` setting applies to TCP-based reverse transports (`tcp`,
  `tcpmux`, `stealth`, `ws`, `wss`, `wsmux`, `wssmux`) and the legacy `[direct]`
  stream engine. It controls the outer TCP connection.
- Reverse `kcp`, `xdi` and `pck` use **KCP MTU**. Raw `udp` and `quic` do not
  use KCP; do not tune them with a KCP setting.
- The current `[l3]` engine has separate **TUN MTU**, automatic path-MTU probing
  and **inner TCP MSS** rules for its carriers, including Spoof and QUIC.
  See [L3 sizing](l3-direct-tunnel.md). `spoof_mtu` limits the outer forged
  packet before fragmentation; it is not the same as the inner MSS.
- It is not part of a [performance preset](performance-presets.md) and a preset
  change leaves it alone. It describes the path the tunnel crosses, not how hard
  the tunnel is being pushed.
- The stream tuning menu accepts 524–1460 bytes (or `0` for kernel default).
  Account for the actual encapsulation overhead when sizing packets; inner
  payload size and outer path MTU are different figures.

---

<div dir="rtl">

## خلاصهٔ فارسی

بیشتر تونل‌ها هیچ‌وقت به این نیاز ندارند. فقط وقتی تنظیمش کن که چیزی مسیر را
اندازه گرفته و همین را گفته باشد.

**مشکلی که حل می‌کند:** بعضی مسیرها نمی‌توانند پکت با اندازهٔ کامل را عبور دهند و
پکت‌های بزرگ را بدون فرستادن پیام ICMP دور می‌ریزند. چون handshake و heartbeat
کوچک‌اند و می‌رسند، تونل وصل می‌شود، وصل می‌ماند و از هر زاویه سالم به‌نظر
می‌رسد — ولی هر انتقال واقعی روی اولین سگمنت کامل گیر می‌کند.

**Manage → Health Check** دقیقاً همین را می‌سنجد و عدد دقیق را چاپ می‌کند.

**تنظیم:** `Manage → Manage Tunnels → Edit → TCP MSS clamp`. مقدار `0` یعنی
انتخاب کرنل در تنظیم `mss` برای اتصال بیرونی TCP. در L3 تنظیم جداگانهٔ
`mss_clamp = 0` از MTU تونل محاسبه می‌شود و `-1` آن را خاموش می‌کند.
**همان مقدار را روی هر دو طرف بگذار** — هر طرف فقط چیزی را که خودش می‌فرستد
clamp می‌کند.

**نکات:** تنظیم `mss` روی اتصال بیرونی TCP اعمال می‌شود
(`tcp`, `tcpmux`, `stealth`, `ws`, `wss`, `wsmux`, `wssmux` و `[direct]` قدیمی).
KCP و حامل‌های KCP از **KCP MTU** استفاده می‌کنند، ولی UDP خام و QUIC، KCP
ندارند. در L3، MTU رابط TUN و clamp برای TCP داخلی جداست؛ `spoof_mtu` هم اندازهٔ
بستهٔ بیرونی را کنترل می‌کند. جزو پریست‌ها نیست و تغییر پریست به آن دست
نمی‌زند. بازهٔ مجاز ۵۲۴ تا ۱۴۶۰ بایت است.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against Backpack v1.8.27.*
