# gitmeta

[![Go Reference](https://pkg.go.dev/badge/github.com/richardwooding/gitmeta.svg)](https://pkg.go.dev/github.com/richardwooding/gitmeta)
[![CI](https://github.com/richardwooding/gitmeta/actions/workflows/ci.yml/badge.svg)](https://github.com/richardwooding/gitmeta/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/richardwooding/gitmeta)](https://goreportcard.com/report/github.com/richardwooding/gitmeta)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Website:** [richardwooding.github.io/gitmeta](https://richardwooding.github.io/gitmeta/)

Fast **per-file git metadata** for Go — last-commit time / author / subject, first-seen,
commit count (churn), and tracked / ignored status — resolved by scanning a working tree
**once** and answering per-path lookups in constant time. **Zero dependencies** (shells out
to the system `git` binary).

The batch design is the point: one `Cache` runs `git ls-files` + a single `git log` pass up
front, so a 10k-file / 5k-commit repo costs **one** git invocation (~½ s) instead of 10k
`git log -1 -- <path>` calls (~100 s).

```sh
go get github.com/richardwooding/gitmeta
```

## One-shot Cache

```go
cache, err := gitmeta.New(ctx, "/path/to/repo")
if err != nil { /* git error */ }
if cache == nil {
    // not a git working tree (or no git binary) — treat as "no git data"
}

if info, ok := cache.Lookup("/path/to/repo/main.go"); ok {
    fmt.Println(info.LastCommitTime, info.LastCommitAuthor, info.CommitCount)
}

cache.IsTracked(path) // bool
cache.IsIgnored(path) // bool
```

`Lookup` returns a `FileGitInfo`:

```go
type FileGitInfo struct {
    LastCommitTime    time.Time
    LastCommitAuthor  string
    LastCommitSubject string
    FirstSeen         time.Time
    CommitCount       int // churn proxy
}
```

Why git rather than filesystem mtimes? A fresh clone sets every file's mtime to checkout
time — so "recently changed" / "hot file" questions need git history, not the filesystem.

## Pool — reuse across calls

A `Pool` keeps one `Cache` per repo and **re-validates on HEAD change**, so repeated lookups
over an unchanging tree don't re-scan. Ideal for a long-running process (server, watcher,
language tooling) that answers many git-metadata queries.

```go
pool := gitmeta.NewPool()
cache, err := pool.Get(ctx, root) // built once per repo, refreshed when HEAD moves
```

## Requirements

- **Go 1.21+**, zero third-party dependencies.
- The system **`git`** binary on `PATH` (`gitmeta.HasGitBinary()` reports its presence;
  `New` returns a nil `Cache` when git is absent or the path isn't a working tree).

## License

MIT — see [LICENSE](LICENSE).

---

Extracted from [file-search-on](https://github.com/richardwooding/file-search-on), where it
powers the `git_*` search attributes and the `hot_files` / `recent_commits` presets.
