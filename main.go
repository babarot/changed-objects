package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

var (
	Version  = "unset"
	Revision = "unset"
)

type CLI struct {
	Option Option
	Stdout io.Writer
	Stderr io.Writer
	Repo   *git.Repository
}

type Option struct {
	Added    bool `long:"added" description:"Return only added objects (defaults: added/deleted/modified)"`
	Deleted  bool `long:"deleted" description:"Return only deleted objects (defaults: added/deleted/modified)"`
	Modified bool `long:"modified" description:"Return only modified objects (defaults: added/deleted/modified)"`
	Dirname  bool `long:"dirname" description:"Return changed objects with their directory name"`

	Version bool `short:"v" long:"version" description:"Show version"`
}

func (o Option) NoKindFlag() bool {
	return !o.Added && !o.Deleted && !o.Modified
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var opt Option
	args, err := flags.ParseArgs(&opt, args)
	if err != nil {
		return 1
	}
	cli := CLI{
		Option: opt,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Repo:   nil,
	}
	if err := cli.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		return 1
	}
	return 0
}

func (c *CLI) Run(args []string) error {
	switch {
	case c.Option.Version:
		fmt.Fprintf(os.Stdout, "%s (%s)\n", Version, Revision)
		return nil
	}

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	r, err := git.PlainOpen(path)
	if err != nil {
		return err
	}
	c.Repo = r

	ref, err := r.Head()
	if err != nil {
		return err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	master, err := c.remoteCommit("origin/master")
	if err != nil {
		return err
	}

	stats, err := c.getStats(master, commit)
	if err != nil {
		return err
	}

	var ss Stats
	if c.Option.Added {
		ss = append(ss, stats.Filter(func(stat Stat) bool {
			return stat.Kind == Addition
		})...)
	}
	if c.Option.Deleted {
		ss = append(ss, stats.Filter(func(stat Stat) bool {
			return stat.Kind == Deletion
		})...)
	}
	if c.Option.Modified {
		ss = append(ss, stats.Filter(func(stat Stat) bool {
			return stat.Kind == Modification
		})...)
	}

	if !c.Option.NoKindFlag() {
		stats = ss
	}

	if c.Option.Dirname {
		stats = stats.Map(func(stat Stat) Stat {
			return Stat{
				Kind: stat.Kind,
				Path: filepath.Dir(stat.Path),
			}
		})
		stats = stats.Unique()
	}

	for _, stat := range stats {
		fmt.Println(stat.Path)
	}

	return nil
}

func computeDiff(from, to *object.Commit) (object.Changes, error) {
	src, err := to.Tree()
	if err != nil {
		return nil, err
	}

	dst, err := from.Tree()
	if err != nil {
		return nil, err
	}

	return object.DiffTree(dst, src)
}

func (c CLI) getStats(from, to *object.Commit) (Stats, error) {
	var err error

	changes, err := computeDiff(from, to)
	if err != nil {
		return nil, err
	}

	var result []Stat
	for _, change := range changes {
		s, err := c.fileStatsFromChange(change)
		if err != nil {
			continue
		}
		result = append(result, s)
	}

	return result, nil
}
