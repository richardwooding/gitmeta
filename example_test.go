package gitmeta_test

import (
	"context"
	"fmt"

	"github.com/richardwooding/gitmeta"
)

// Example shows the one-shot Cache: scan a working tree once, then answer
// per-file metadata lookups in constant time. (Output is omitted because
// commit times/authors are repo-specific.)
func Example() {
	ctx := context.Background()

	cache, err := gitmeta.New(ctx, "/path/to/repo")
	if err != nil {
		panic(err)
	}
	if cache == nil {
		fmt.Println("not a git working tree")
		return
	}

	if info, ok := cache.Lookup("/path/to/repo/main.go"); ok {
		fmt.Printf("last touched %s by %s — %d commits\n",
			info.LastCommitTime.Format("2006-01-02"),
			info.LastCommitAuthor,
			info.CommitCount)
	}
}

// ExamplePool shows reusing one Cache per repo across many calls; the
// pool re-validates on HEAD change, so an unchanged tree is scanned once.
func ExamplePool() {
	pool := gitmeta.NewPool()
	ctx := context.Background()

	for _, root := range []string{"/repo/a", "/repo/b", "/repo/a"} {
		cache, err := pool.Get(ctx, root) // "/repo/a" the 2nd time is a cache hit
		if err != nil || cache == nil {
			continue
		}
		_ = cache.IsTracked(root + "/README.md")
	}
}
