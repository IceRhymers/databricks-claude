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
