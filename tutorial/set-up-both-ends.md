# Set up both ends from one server

Use this after creating **one end** with Setup Iran or Setup Kharej. Backpack
can reach the second server over SSH, install Backpack there if needed, mirror
the tunnel configuration, and start its service. You can also start in the web
panel at **Servers → Add a server**, then **Tunnels → Add tunnel**.

## From the CLI

1. Run `sudo backpack` on the first machine and create its end. At the end of
   setup, answer **yes** to **Set up the other server over SSH now**. For an
   existing tunnel, choose **Manage → Set up the other server** and select it.
2. At **This server's address as the peer reaches it**, enter the **real,
   reachable IP or hostname of the machine you are on**, never a forged source
   IP. In reverse mode this is usually Iran; in direct mode it is usually
   kharej. A listener bound to `0.0.0.0` still needs its public address here.
3. If asked for **Ports to expose on Iran**, enter the ports users connect to,
   such as `2095`. For a reverse tunnel whose ports exist only on the Iran end,
   use the full mapping if the actual service runs elsewhere, for example
   `2095=127.0.0.1:2096`.
4. At **Where is the other end?**, select an existing managed server or **Add a
   server**. For a new server enter its SSH address, port, username and
   password. The password is hidden while typed. The address here is the
   **other** machine, not the public address from step 2.
5. Read the local/remote names shown before **Create or update the other end
   now**. If a tunnel with that name already exists remotely, Backpack asks
   before replacing its settings. Confirm only for the intended tunnel.

Downloading and installing on a fresh server can take several minutes. If the
first attempt fails, the local tunnel remains in place; fix the error and run
**Manage → Set up the other server** again. The panel and CLI use the same SSH
setup route and a 20-minute installer timeout.

**Both tunnel services are running** confirms the processes started. A separate
peer-connection check follows, but even that is not a test of your application:
send traffic to an exposed port and read **Tunnel Metrics** on both ends.

## If the server was rebuilt

A rebuild usually changes the SSH host key. Backpack refuses a saved server
whose key changed. First verify the new fingerprint through the server's
console or another trusted route. Then remove that server's **fleet entry** in
the web panel at **Servers → Remove**, add it again, and repeat the CLI pairing.
Removing the fleet entry does not delete tunnels or services on either host;
it does forget this panel's pairing record.

On a server card, the **bold name** is the name you gave it in Backpack; the
smaller name below is the hostname reported by that machine. They can differ
after a rebuild, and neither is proof that a deleted entry came back.

## If setup does not pass traffic

| Observation | Check |
|---|---|
| SSH host key changed | Verify the rebuild and fingerprint, then remove and re-add the fleet entry. Do not disable host-key checking. |
| Install failed | Read the installer stage and exit status; check the target's access to GitHub, free space and root login. You can retry without recreating the local tunnel. |
| Services running, peer not verified | Compare carrier, token, real peer IP, tunnel port and firewall on the listening side. Read logs on both servers. |
| Peer connected, forwarded port fails | Confirm the service actually listens on the mapped address and port on kharej. |

See [managed servers](../docs/managed-servers.md) for the panel's security and
pairing model, and [Before you start](before-you-start.md) for ports and roles.

---

<div dir="rtl">

## خلاصهٔ فارسی

بعد از ساخت سمت اول، به سؤال **Set up the other server over SSH now** جواب `y`
بده؛ برای تونل موجود از **Manage → Set up the other server** برو. آدرس سؤال
اول، **آی‌پی واقعی همین سرور** از دید سمت مقابل است. در **Where is the other
end?** سرور دوم را انتخاب یا با مشخصات SSH اضافه کن. اگر برای نصب زمان زیادی
صرف شد صبر کن؛ نصب‌کننده تا ۲۰ دقیقه فرصت دارد. در پایان، علاوه بر روشن بودن
هر دو سرویس، عبور واقعی ترافیک از پورت فورواردشده را بررسی کن.

اگر سرور دوم ریبیلد شده و کلید SSH عوض شده، ابتدا اثرانگشت جدید را از کنسول
مطمئن بررسی کن؛ سپس فقط **مدخل سرور** را از Servers حذف و دوباره اضافه کن.
حذف مدخل، تونل‌های روی سرورها را پاک نمی‌کند. نام پررنگ کارت، نام ثبت‌شده در
Backpack است؛ نام کوچک زیر آن، hostname خود سیستم‌عامل است.

</div>

---
[← Back to the tutorials](README.md)
