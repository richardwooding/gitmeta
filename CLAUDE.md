# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`gitmeta` is a zero-dependency Go library that resolves **per-file git metadata** (last-commit time/author/subject, first-seen, commit count, tracked/ignored status) by scanning a working tree once and answering per-path lookups in constant time. It shells out to the system `git` binary — no go-git or other third-party deps. Extracted from [file-search-on](https://github.com/richardwooding/file-search-on), where it powers `git_*` search attributes.

## Commands

```sh
go build ./...
go vet ./...
go test ./...                              # all tests
go test -race ./...                        # CI runs with -race
go test -run TestNew_SingleCommit ./...    # single test by name
go test -coverprofile=coverage.out ./...   # coverage
```

Lint locally with `golangci-lint run` (CI uses `golangci-lint-action`, latest version). The go.mod floor is **Go 1.21**; CI builds/tests against both `1.21` and `stable`.

Tests create throwaway git repos in `t.TempDir()` and make real commits, so a git identity must exist. The test helpers set a per-repo identity, but on a bare CI runner a global one is also configured (`git config --global user.email/user.name`). Tests `t.Skip` when no git binary is on PATH.

## Architecture

The whole design rests on **batching git invocations**. The naive approach — `git log -1 -- <path>` per file — is O(files) subprocesses (~100s on a 10k-file repo). Instead `New` runs a fixed handful of git commands once and builds in-memory maps, making each subsequent `Lookup`/`IsTracked`/`IsIgnored` a map read.

Two types, two files:

- **`gitmeta.go` — `Cache`**: the per-repo scan result, effectively immutable after construction and safe for concurrent reads. `New(ctx, root)` runs `rev-parse --show-toplevel`, `rev-parse HEAD`, two `ls-files` calls (tracked; others+ignored), and one `git log --name-only` pass. `logFiles` parses the log newest-first: the first appearance of a path fixes `LastCommit*`, every appearance overwrites `FirstSeen` (so the oldest wins) and increments `CommitCount`.

- **`pool.go` — `Pool`**: caches one `Cache` per canonical repo root, keyed by `git rev-parse --show-toplevel`. `Get` re-runs `rev-parse HEAD` on every call (~3–5ms) and rebuilds only when HEAD moved. Built for a long-running process (MCP server, watcher) that answers many queries against a mostly-static tree.

### Two invariants that pervade the code

1. **nil `Cache` means "no git data", not an error.** `New` returns `(nil, nil)` when `root` isn't in a git tree or `git` is absent — this is the *common, expected* path for non-repo trees, and callers must handle it as "leave git fields at zero values." Hard errors (`(nil, err)`) are reserved for a present-but-broken git (subprocess crash, ctx cancel). All read methods are nil-receiver-safe. An empty repo (init, no commits) yields a non-nil cache with empty file metadata via `buildEmptyHeadCache` so tracked/ignored still answer.

2. **Dual-root path resolution for the macOS symlink case.** git canonicalizes `/tmp/...` to `/private/tmp/...`, but a caller's `filepath.Walk` often emits the symlinked form. `Cache` stores both `repoRoot` (git's canonical view) and `repoRootAlt` (the as-supplied absolute root). `toRel` tries both prefixes — an alloc-free comparison — rather than paying an `EvalSymlinks` stat per file. Keys throughout are repo-relative forward-slash paths (the form `ls-files` emits).

`ctx` propagates to every subprocess via `exec.CommandContext`, so a cancelled walk tears down the git processes; `runGit` passes ctx-cancellation errors through unwrapped (for `errors.Is`) and wraps everything else with stderr.
