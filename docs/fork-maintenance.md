# admin6501 fork maintenance

The product and executable remain BackPack / `backpack`. Installation, release
checks, update downloads, issue reports and support links use
`https://github.com/admin6501/BackPack`. Existing service and configuration paths
remain compatible. Upstream copyright and provenance notices remain in the legal
and About surfaces. Upstream channel, group, website and donation promotions have
been removed from the installer, menus, bot, panel and documentation.

## Publishing signed releases

This fork cannot sign with the upstream publisher's private key. Generate an
Ed25519 signing pair with `make release-key` in a trusted terminal. Store the
private half as the repository's GitHub Actions secret `RELEASE_SIGNING_KEY`;
never commit it. Keep a protected backup of that secret for future releases.

`make release` requires that secret in its environment. It derives the public
half, embeds it in every architecture's binary, signs `SHA256SUMS` and publishes
`SHA256SUMS.sig` with the release assets. The existing Release workflow invokes
this target. Missing, malformed or internally inconsistent keys stop the build.

A plain source build has no pinned release key and refuses automatic binary
updates. For a source build that should accept your signed releases, build with
`make build RELEASE_PUBLIC_KEY=<base64-public-key>` using your verified public
key. Never use the private half in this argument. Keep the same signing key for
future releases: changing it requires a deliberate trust migration on installed
servers. The first installation still relies on the installer and published
checksums; this change does not add a separate signature verifier to install.sh.

No release is created by this source change. Before publishing, configure the
secret, advance VERSION and app.Version together, then use a matching release tag.

## Fixed regressions

- Linux service management invokes lowercase `systemctl`.
- Password changes and credential-bearing backups require administrator scope.
- Password changes accept the panel's JSON and legacy form requests.
- IPv6 backend destinations use host/port parsing rather than colon splitting.
- UDP destination learning happens after packet authentication, including per-path
  routing with multipath and FEC; unauthenticated datagrams cannot repin a path.
- QUIC listeners can close while waiting for their first connection.
- QUIC stream closure interrupts readers and writers; graceful write completion
  remains separate so outstanding response data can drain.
- UDP detection survives bandwidth wrapping and skips the PROXY header.
- Expired UDP pairing attempts remove their flow and release the connection slot.
- Idle TCP pool connections close when their owning context is cancelled.
- TCP EOF propagates a write half-close where the transport supports it, preserving
  responses after request EOF. Cancellation and errors still close both sides.
- SSH handshake timeouts and cancellation cover the handshake, not just TCP dial.

Regression tests exercise loopback sockets, authentication scopes, cancellation,
response completion and signed-release key validation. Protocols that do not
support half-close still use full closure on EOF.
