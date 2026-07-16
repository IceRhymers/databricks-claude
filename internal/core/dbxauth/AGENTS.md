# internal/core/dbxauth

Databricks-CLI OAuth token fetch, workspace host discovery, and the gateway-URL
join — shared by all three launchers. Created in #218.

Parent: [`internal/core`](../) · Module: `github.com/IceRhymers/databricks-agents`

## Key files

| File | Purpose |
|---|---|
| `dbxauth.go` | `Config`, `Fetcher`, `NewFetcher`, `NewProvider`, `DiscoverHost`, `GatewayURL`, plus the unexported `tokenResponse`/`authEnvResponse` wire shapes and `parseTokenResponse`/`expiryTime`. |
| `dbxauth_test.go` | Helper-binary tests (no `exec.Command` mocking, per CLAUDE.md), including the two `HonorsDATABRICKS_CLI` bugfix tests. |

## Why this package exists

Before #218 each launcher carried its own byte-identical copy of this logic in
`cmd/databricks-*/token.go` (~85 of ~130 lines each). The copies had drifted:

- **Argument order.** claude took `(profile, cmdName)`; codex and opencode took
  `(cmdName, profile)`. Both are `(string, string)`, so a wrong-order call site
  compiled silently.
- **CLI resolution.** Only claude routed through `internal/core/cli`. codex and
  opencode exec'd the bare name, so under launchd/systemd's minimal PATH their
  token fetch could fail to find a CLI their own `authcheck` path had just
  resolved. That was a real bug, fixed here.

## Rules

**Use keyed `Config` literals.** Both fields are strings. Keying makes a
Profile/CLIPath swap conspicuous; it does **not** make it impossible, and no
option available at this scope does — a named-type (`type Profile string`)
conversion happens at the call site, which is where the swap happens, so it just
relocates the problem. True inexpressibility would need origin-typing every
producer (flag targets, the launchers' state structs, authcheck's params), which
is out of scope. Because there is no compiler backstop here, the credential
helper — the one caller passing a non-empty `CLIPath` — is pinned by a
behavioral test instead (`TestCredentialHelper_HonorsStateDatabricksCLIPath`).
Shape mirrors `updater.Config` (#217).

**Per-tool gateway paths stay launcher-side.** This package joins a host and a
path; it never learns which path a tool uses. That is structural, not stylistic:
`profile.Profile.GatewayPath` is single-valued and opencode has **two** upstreams
(Anthropic + Gemini Native), so a per-tool path value cannot represent it. Some
call sites (e.g. opencode's `config_cmd.go`) also want a gateway URL without
constructing a `Profile` at all.

## Known asymmetry: the MDM tier is process-global

`cli.ResolveDatabricksCLI` consults an MDM-managed `databricksCliPath` through a
**package-level reader** wired by `cli.SetMDMReader`. Only `databricks-claude`
wires it (`cmd/databricks-claude/main.go`, hoisted above every early-exit
dispatcher so all entry points see the real reader). The default is a no-op, so
codex and opencode get the `$DATABRICKS_CLI`, `PATH`, and fallback-dir tiers but
skip MDM.

**This is correct and deliberate.** The MDM domain is `com.icerhymers.databricks-claude`
and the key is provisioned for Claude Desktop's credential-helper flow; codex and
opencode have no MDM/`.pkg`/trust-profile surface. Wiring them to a claude-branded
domain would be a scope error.

**It is also not new.** `authcheck` already resolves through the same global from
five codex/opencode call sites that never wire a reader. `dbxauth` widens that
coupling; it did not create it. The fix belongs with the `authcheck` alignment
below (#227), which owns both packages — fixing it in `dbxauth` alone would make
it diverge from `authcheck`, which is the problem that issue exists to close.

**Fragility to know about:** the ordering invariant (every CLI-resolving entry
point dispatches *below* `SetMDMReader`) is currently held only by a comment in
`main.go`. A future early-exit added above that line would silently degrade the
MDM tier to the no-op reader.

## Tracked debt

`authcheck` and `dbxauth` are siblings in `internal/core`: both shell the
Databricks CLI, both carry the same `(profile, cliPath)` pair, both resolve
through `internal/core/cli` — with **opposite conventions** (`authcheck` is
positional, `dbxauth` is keyed). Aligning them (`IsAuthenticated(dbxauth.Config)`,
or folding authcheck in) is tracked in **#227** and deliberately not absorbed into
#218: authcheck has four live non-empty-`cmdName` consumers (`setup.go:65,84`,
`desktop_config.go:337`, `serve_install.go:289`), which makes it a materially
larger change. #227 also owns making the MDM reader explicit.
