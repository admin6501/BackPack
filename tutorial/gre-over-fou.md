# Set up GRE over FOU

Update Backpack on both Linux servers and run `sudo backpack` as root. This
carrier uses the real server addresses and requires UDP to reach the listening
end. It carries authenticated, encrypted GRE; it is not compatible with a plain
Linux kernel GRE-over-FOU endpoint.

## Choose the direction

On each machine choose its geographic side: **Setup Iran** or **Setup Kharej**.
Then choose the same direction on both:

| Direction | Menu | Iran | Kharej |
|---|---|---|---|
| Reverse | Reverse → 4) GRE → 1) GRE over FOU | listens | dials Iran |
| Direct | Direct → GRE over FOU | dials kharej | listens |

The direction is not asked again. The web panel uses the same family/carrier
placement. Existing tunnels keep their settings; this is a setup menu change.

## Fill in the tunnel

1. On the listening end enter the UDP tunnel port and allow it through that
   server's firewall. On the dialing end enter the listener's reachable real
   IP or domain and the same port.
2. Keep the suggested unused private subnet unless it conflicts with your
   routes. In the default pair Iran uses `10.10.0.1/30` and peer `10.10.0.2`;
   kharej uses `10.10.0.2/30` and peer `10.10.0.1`. Only the local address has
   the prefix. A later tunnel may receive a different unused subnet.
3. Enter at least one forwarded port on Iran, for example `2082, 2095`.
   Kharej hosts the corresponding services and has no separate forwarded-port list.
4. Use the same token on both ends. In manual GRE setup kharej suggests a token;
   Iran accepts that token or generates one if kharej has not been configured yet.
5. Review the settings and create the tunnel.

## Complete the other end

Copy the generated link into **Setup from a link** on the other server. The
link carries the carrier, token, opposite mode and swapped private addresses.
A link produced on kharej asks for the forwarded ports when imported on Iran.
Treat the link as a secret because it contains the token.

Alternatively use **Manage → this tunnel → Set up the other server** and enter
or select the SSH destination. The paired setup uses the same mirrored values.
The web panel's managed-server setup also writes the opposite mode on the peer.

Check that both services are running and test one forwarded service. In either
direction users connect to the forwarded port on Iran.

## فارسی

پس از انتخاب Setup Iran یا Setup Kharej، برای ریورس **Reverse → خانوادهٔ ۴:
GRE → گزینهٔ ۱: GRE over FOU** و برای دایرکت **Direct → GRE over FOU** را بزنید.
در ریورس خارج به ایران وصل می‌شود؛ در دایرکت ایران به خارج. پورت کاربران همیشه
روی ایران است. لینک ستاپ یا گزینهٔ نصب سمت مقابل با SSH، حالت طرف مقابل و
آدرس‌های خصوصی جابه‌جاشده را منتقل می‌کند. در وب‌پنل هم همین چیدمان برقرار است.

[GRE over FOU reference](../docs/gre-over-fou.md) · [All tutorials](README.md)
