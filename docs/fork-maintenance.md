# admin6501 fork maintenance

The product and executable remain BackPack / `backpack`. Installation, release
checks, update downloads, issue reports and support links use
`https://github.com/admin6501/BackPack`. Existing service and configuration paths
remain compatible. Upstream copyright and provenance notices remain in the legal
and About surfaces. Upstream channel, group, website and donation promotions have
been removed from the installer, menus, bot, panel and documentation.

## Building AMD64 and ARM64 without a signing key

No GitHub secret is required. Install the Go version specified by go.mod and
run these commands from the repository root:

```sh
make release ARCHES="amd64 arm64" ARMS=""
```

This cross-compiles Linux binaries for both x86-64 (Intel/AMD) and ARM64 on the
same build machine. Output binaries are `dist/backpack-linux-amd64` and
`dist/backpack-linux-arm64`. Release archives are
`release/backpack_linux_amd64.tar.gz` and `release/backpack_linux_arm64.tar.gz`;
each contains the `backpack` executable. Upload the archives and
`release/SHA256SUMS` to a GitHub Release for the installer and updater to use.

The Release workflow builds all supported architectures automatically on a
matching `v*` tag, including these two. Advance VERSION and app.Version together
before tagging. The code must first be merged into the branch you tag. This PR
does not create a release or deploy anything.

## Optional release signatures

Without `RELEASE_SIGNING_KEY`, builds and updates work using SHA256 checksums.
These detect a mismatched/corrupted archive but do not independently authenticate
the publisher when both the archive and its checksum list are replaced.

Signing can still be enabled later: generate a key pair using `make release-key`
in a trusted terminal and store the private half as the repository secret
`RELEASE_SIGNING_KEY`. Never commit it. `make release` derives and embeds the
public half and signs SHA256SUMS. A malformed configured key fails the build.
An unsigned build removes any stale SHA256SUMS.sig from an earlier build.

Binaries carrying a public key continue to require valid signatures. Removing a
secret does not downgrade already installed signed builds; switch those installs
to a checksum-only build manually if that is intended. For source builds,
`make build RELEASE_PUBLIC_KEY=<base64-public-key>` enables signature checks.

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
