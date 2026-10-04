# Backup & restore

Everything in one portable `.tar.gz`: every tunnel and token, the web-panel
password, Telegram settings, TLS certificates, and the auto-refresh schedule.
Backups live in `/root/BackPack/backups`.

## Restoring onto a different machine

A backup carries everything except one thing: the key that decrypts the stored
root passwords of your **managed servers**. That is deliberate — a backup is a
file people move, and without the seal every copy of it would carry those
passwords in the clear to wherever it went.

Restoring onto the same machine is unaffected: the key is already there and the
restore leaves it alone.

Restoring onto a **different** machine brings the fleet list back without the
credentials, and the panel asks for each server's password again. That is the
right default. If you would rather not re-enter them — the panel machine died
and this is the recovery — keep the key yourself, separately:

**On the machine that has the fleet, before you need it:**

`sudo backpack` → **Backup & Restore** → **Show the fleet key**

Store it somewhere the backup is not. Keeping them together undoes the only
thing sealing them achieves.

**On the new machine, after restoring the backup:**

`sudo backpack` → **Backup & Restore** → **Restore the fleet key**

It refuses if that machine already has a key of its own, because overwriting one
would make every password currently sealed there unreadable and there is no
undo — move the existing key aside yourself if that is genuinely what you mean.

Both entries only appear when there is actually a managed server with a sealed
password, so a single-machine install never sees them.


## Restore

Restoring **re-registers and starts every tunnel**, and traffic totals carry on
from where the backup left off rather than resetting to zero.

## Where you can do it

- the **CLI** — **Backup & Restore**,
- the [web panel](web-panel.md) — **Settings**, or
- the [Telegram bot](telegram-bot.md) — **Backup** button.

> Keep a backup file private — it contains tokens and the panel password.

---

<div dir="rtl">

## خلاصهٔ فارسی

همه‌چیز در یک فایل `.tar.gz` قابل‌حمل: تمام تونل‌ها و توکن‌ها، رمز پنل وب،
تنظیمات تلگرام، گواهی‌های TLS و زمان‌بندی ری‌فرش خودکار. فایل‌ها در
`/root/BackPack/backups` ذخیره می‌شوند.

**بازگردانی** همهٔ تونل‌ها را دوباره ثبت و استارت می‌کند و آمار ترافیک از همان
جایی که بوده ادامه پیدا می‌کند، نه از صفر.

از سه جا می‌شود انجامش داد: منوی CLI (گزینهٔ Backup & Restore)،
[پنل وب](web-panel.md) در بخش Settings، یا دکمهٔ Backup در
[ربات تلگرام](telegram-bot.md).

> فایل پشتیبان را خصوصی نگه دار — توکن‌ها و رمز پنل داخلش است.

</div>

---
[← Back to the docs index](README.md)

## Getting a backup off the machine

Backups are written to `/root/BackPack/backups` — on the server they describe.
The case they exist for is the case where that server is gone, so a copy
somewhere else is the only one that will be there.

**Backup & Restore → Copy backups off this machine.** It takes a command, with
`{}` standing for the backup file's path:

```
rclone copy {} remote:backpack/
scp {} backup@10.0.0.9:/srv/backpack/
restic backup {}
```

A command rather than a list of providers, because whatever you already use to
move files is the thing that works and the thing that does not need a release
to support a new destination.

It runs **directly, not through a shell**, so a `;` or a `|` in it is a word
rather than syntax — a destination read from a file cannot become a second
command. For a pipeline, write `sh -c '...'` and the decision is visible to
anyone reading the config.

The screen offers to try it immediately with your newest backup. Take the
offer: a destination that does not work fails the same way every week, and
without this you find out much later.

The weekly automatic backup sends a copy too, and reports the two separately.
A backup that could not be *written* is a full disk; a copy that could not be
*sent* is a destination that has changed. They have different fixes, so rolling
them into one message would send you to look at the wrong one.

## Testing a restore without performing one

A recovery procedure that has never been run is the ordinary state of a
disaster-recovery plan, and it is why they fail: the first time anybody
exercises it is the day it has to work, on a machine that is already gone, with
whatever the backup turned out not to contain.

**Backup & Restore → Test a restore.** It stages the archive exactly as a real
restore would — the same reader, the same name checks, the same size limits,
the same refusals, with the same wording — into a scratch directory, tells you
what it holds, and throws it away. **Nothing on the machine changes.**

It reports the tunnels, the panel settings, the Telegram configuration and the
certificates it would put back, and it warns about the two things that are
silent otherwise:

- a backup with **no tunnel configuration at all**, which would restore a
  machine with nothing on it;
- managed servers whose passwords are **sealed with a key the archive does not
  contain**. That is deliberate — a stolen backup must not carry a fleet with
  it — and it means restoring onto a *different* machine gives you the server
  list and no way to reach any of them. Take the fleet key now and keep it
  somewhere else; the drill is where you find that out while it is still cheap.

Run it after any change to what the machine holds, and once before you need it.

## Restoring onto another server over SSH

**Backup & Restore → Restore on another server over SSH** (option 5) picks an
existing backup or a custom archive path, then asks for the destination IP,
SSH port (default 22), username (default root), and a hidden password. The
destination must run Linux with coreutils and allow root login or passwordless
`sudo` for the selected user.

Before changing the destination, Backpack validates a private snapshot of the
archive, checks SSH and privileges, and displays the destination, SSH host-key
fingerprint and backup contents for confirmation. Cancel leaves destination
settings untouched. The password is used only for this connection, never
stored in the managed-server list or placed in command arguments.

After confirmation it keeps an installed Backpack that supports automatic
restore. If Backpack is absent, or its version predates the automatic restore
command, it runs the official installer to install/update it first. The installer
is pinned to an audited commit and its SHA-256 is checked before execution. It transfers
the archive over the same encrypted SSH connection to a private temporary file,
checks SHA-256, and runs the existing restore operation. Uploaded and local
temporary archives are removed on completion or failure. If the connection is
lost during restore, inspect the destination before retrying: it may already
have applied the settings.

Matching configurations are overwritten and tunnels restarted; destination
configurations not named in the backup remain. A failed tunnel restart or web
panel restart is reported as a partial restore, never as full success.
IP addresses and tunnel settings remain exactly as saved. Orphan metrics and
configuration-history files without a matching tunnel config are excluded from
new backups and removed from the staged restore, including older archives.
Active tunnel counters and histories are retained.

Traffic totals, quota limits, customer links/settings, country labels and
history are restored as recorded **when the backup was created**. The CLI
stops active tunnel, panel and monitor writers before applying the staged
configuration, preventing their final flushes from overwriting archived usage.
Unarchived destination data keeps its final counters. Previously active
services are resumed; inability to stop a writer aborts before the commit, and
failure to resume a service is reported as partial recovery. Sealed fleet passwords still need the separately kept
fleet key; this option does not transfer that key.

For automation on the destination:

```sh
backpack backup capabilities --json
backpack backup check /path/to/backup.tar.gz --json
sudo backpack backup restore /path/to/backup.tar.gz --yes --json
```

`capabilities` advertises `restore_protocol: 2`. `check` validates without
changing settings. `restore` requires root and `--yes`; exit code 4 means the
settings were applied but a restored service failed to start.

<div dir="rtl">

از **Backup & Restore → گزینهٔ ۵، Restore on another server over SSH** بکاپ
را انتخاب کنید و آی‌پی، پورت SSH، نام کاربری و رمز سرور مقصد را وارد کنید.
رمز مخفی وارد می‌شود و ذخیره نمی‌شود. بعد از بررسی اتصال و نمایش مشخصات
مقصد، ریستور را تأیید کنید. اگر بک‌پک نصب نباشد نصب می‌شود؛ نسخه‌های قدیمی
که فرمان ریستور خودکار ندارند، ابتدا به‌روزرسانی می‌شوند. فایل به‌صورت امن
منتقل و صحت آن بررسی می‌شود، سپس تنظیمات ریستور و تونل‌ها ری‌استارت می‌شوند.
نام کاربری باید root باشد یا دسترسی sudo بدون رمز داشته باشد. آدرس‌ها و تنظیمات تونل مطابق بکاپ باقی می‌مانند.
مصرف و تاریخچه تا زمان ساخت بکاپ، سقف ترافیک و اطلاعات کاربران حفظ می‌شوند.
سرویس‌های نویسندهٔ مصرف پیش از اعمال بکاپ متوقف و سپس دوباره فعال می‌شوند تا
شمارنده‌های قدیمی مقصد، مصرف ریستورشده را بازنویسی نکنند. کلید fleet جداگانه
باید نگهداری و در صورت نیاز روی مقصد وارد شود.

</div>

---

*Last verified against Backpack v1.8.24.*
