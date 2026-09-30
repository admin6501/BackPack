# Contributing

## Before anything else: the licence

Backpack is **AGPL-3.0**, and `NOTICE` records that part of the data plane
derives from prior AGPL/GPL work. That has three consequences worth knowing
before you write anything:

- The project cannot be relicensed, by anyone, including its maintainer.
- AGPL §13 applies: if you run a modified version as a network service, the
  people using it are entitled to your source.
- `NOTICE` carries two additional terms permitted by AGPL §7. They ask that the
  attribution line be kept, and they decline to license the name. `TRADEMARK.md`
  says exactly what that covers — in short: **fork freely, rename if you
  redistribute**, and nominative use ("compatible with Backpack") is always
  fine.

You keep the copyright on what you write. There is no CLA.

## Running checks locally

Use the Go version in `go.mod` (currently 1.26.6). Node.js 22 is used in CI
for the web panel tests; without Node.js, those tests are skipped locally.
Before opening a code change, run the relevant checks:

```sh
gofmt -l .                      # must print nothing
go vet ./...
go test ./... -timeout 20m
go test ./... -race -timeout 20m # the transports are concurrent
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

CI also runs fuzz tests, cross-builds, a sustained traffic test, and a wire
compatibility test against the previous release. Coverage is measured with
`-coverpkg=./...`; both the overall and per-package floors are checked in
`.github/workflows/ci.yml`. If coverage drops, investigate what stopped being
exercised before changing a floor. For documentation-only changes, verify the
links and examples you changed; the pull request still runs CI.

## How this codebase is written, and why it will look unusual

**The comments carry the institutional memory.** They say what was tried, what
it cost, and why the obvious alternative is wrong. Several bugs in this project
were found *because* a comment stated an invariant precisely enough to check.

When you change something that a comment explains, change the comment. When you
fix a bug, write down what it did — not "fixed nil check" but what a person
saw: which message, which screen, what they concluded from it. The next person
to read that code is usually trying to work out why it is the way it is.

**Tests are named after the behaviour, not the function.**
`TestAPairingThatIsNeverAnsweredTimesOutAndGivesBackItsSlot`, not `TestRun`. A
test's name should tell you what capability exists.

**A guard that scans source is a legitimate test here.** Several exist —
`TestEveryConfigKeyReachesCodeThatActsOnIt`, `TestEveryPanelAPIFunctionIsCalled`,
`TestEveryPairingLoopTimesOutAndCleansUpOnShutdown`. They catch the class of bug
this codebase actually produces: a feature wired on one side only. Three unwired
entry points were found by hand before the last of those existed.

## Things to know before you touch them

- **Both ends must match.** Dozens of settings are paired, and a mismatch
  produces the worst failure this system has: the tunnel comes up, reports
  itself connected, and carries nothing. If you add a setting that both ends
  read, add it to the peer mirror and say so in the docs.
- **The data path is `internal/tunnel/l3`, `internal/server/transport`,
  `internal/client/transport` and `internal/utils/handlers`.** It has allocation
  budgets and benchmark gates. A change there that adds an allocation per packet
  fails the build, deliberately.
- **`config/` is the public API.** Everything else is `internal/` and free to
  move. A new `toml:` key must be read by something, and a test proves it.
- **Never break the wire without a plan.** `internal/e2e/wirecompat_test.go`
  runs the previous release against this one in both directions. Protocol
  version negotiation exists (`internal/tunnel/l3/version.go`); use it.

## Opening a change

- Keep each pull request focused. Include related fixes and tests in the same
  change so the behaviour can be reviewed together.
- Say what a user would have seen before your change and what they see after.
- For new behaviour or a bug fix, test the outcome at a useful boundary: a CLI
  action, panel API, configuration load, or traffic across a tunnel. Include
  reproduction steps when the result depends on a particular network or host.
- Structural changes go in one sub-package or one transport at a time, never as
  one sweep. `internal/manage` is being split that way on purpose; see
  `internal/manage/core/doc.go` for why the seam was cut where it was.

## Reporting a security issue

Do not open an issue. See `SECURITY.md`.
