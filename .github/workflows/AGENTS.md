<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# workflows

## Purpose
GitHub Actions CI pipeline. Runs on pull requests to `master`.

## Key Files

| File | Description |
|------|-------------|
| `ci.yml` | Single CI job: checkout, setup Go (version from `go.mod`), run `go test ./... -v`, `go vet ./...`, and `go build` |

## For AI Agents

### Working In This Directory
- The CI job runs on `ubuntu-latest`. Tests that use `syscall.Flock` work on Linux.
- Go version is read from `go.mod` (currently Go 1.22).
- If adding new CI steps, keep them in the single `test` job unless parallelism is needed.

### Testing Requirements
- CI must pass: `go test ./... -v`, `go vet ./...`, and `go build`.

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->

## Release Process (release-please)

Releases are automated via [release-please](https://github.com/googleapis/release-please). It watches commits merged to `master` and opens a release PR automatically — **but only when commits follow the Conventional Commits format**.

### Commit prefix rules

| Prefix | Version bump | When to use |
|--------|-------------|-------------|
| `feat:` | minor (0.X.0) | New user-facing feature or behaviour change |
| `fix:` | patch (0.0.X) | Bug fix |
| `feat!:` / `fix!:` / `BREAKING CHANGE:` | major (X.0.0) | Breaking API or behaviour change |
| `chore:`, `docs:`, `refactor:`, `test:` | none | Internal — **will NOT trigger a release PR** |

### Rules for AI agents

- **Every PR that changes user-facing behaviour must include at least one `feat:` or `fix:` commit.** Without it, release-please skips the run and no release PR is opened.
- Squash-merge is fine — the squash commit message is what release-please reads.
- `chore:` is safe for housekeeping (formatting, test updates, doc-only changes) but will not produce a release.
- If a release PR is unexpectedly missing after a merge, check the release workflow logs: the most common cause is a non-conventional commit message.

### Commit message requirements (release-please)

- **All implementation commits MUST start with `feat:` or `fix:`.** release-please reads commit messages on master — the branch name prefix (`feat/`) does NOT trigger a version bump. Without a conventional prefix the release PR is never opened.
- `chore:`, `docs:`, `refactor:`, `test:` prefixes will NOT trigger a release PR.
