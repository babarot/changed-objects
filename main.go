package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jessevdk/go-flags"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
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
	Added    bool `long:"added" description:"Return added objects"`
	Deleted  bool `long:"deleted" description:"Return deleted objects"`
	Modified bool `long:"modified" description:"Return modified objects"`

	Version bool `short:"v" long:"version" description:"Show version"`
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

	master, err := c.originMasterCommit()
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
			return stat.Kind == "insert"
		})...)
	}
	if c.Option.Deleted {
		ss = append(ss, stats.Filter(func(stat Stat) bool {
			return stat.Kind == "delete"
		})...)
	}
	if c.Option.Modified {
		ss = append(ss, stats.Filter(func(stat Stat) bool {
			return stat.Kind == "modify"
		})...)
	}

	for _, stat := range stats {
		fmt.Println(stat.Path)
	}

	return nil
}

func (c CLI) originMasterCommit() (*object.Commit, error) {
	r := c.Repo

	refs, err := r.References()
	if err != nil {
		return nil, err
	}

	var cmt *object.Commit
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name() == "refs/remotes/origin/master" {
			commit, err := r.CommitObject(ref.Hash())
			if err != nil {
				return err
			}
			cmt = commit
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cmt, nil
}

func masterCommit(r *git.Repository) (*object.Commit, error) {
	branchRefs, err := r.Branches()
	if err != nil {
		return nil, err
	}

	var c *object.Commit
	err = branchRefs.ForEach(func(branchRef *plumbing.Reference) error {
		if branchRef.Name() == "refs/heads/master" {
			commit, err := r.CommitObject(branchRef.Hash())
			if err != nil {
				return err
			}
			c = commit
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Stat represents the stats for a file in a commit.
type Stat struct {
	Kind string
	Path string
}

type Stats []Stat

func (ss *Stats) Filter(f func(Stat) bool) Stats {
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if f(stat) {
			stats = append(stats, stat)
		}
	}
	// *ss = stats
	return stats
}

func (c CLI) getStats(from, to *object.Commit) (Stats, error) {
	var err error
	if to.NumParents() != 0 && from == nil {
		from, err = to.Parent(0)
		if err != nil {
			return nil, err
		}
	}

	if from == nil {
		return fileStatsFromCommit(to)
	}

	return c.fileStatsFromDiff(from, to)
}

func fileStatsFromCommit(c *object.Commit) ([]Stat, error) {
	var result []Stat
	files, err := c.Files()
	if err != nil {
		return nil, err
	}

	err = files.ForEach(func(f *object.File) error {
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c CLI) fileStatsFromDiff(from, to *object.Commit) ([]Stat, error) {
	changes, err := computeDiff(from, to)
	if err != nil {
		return nil, err
	}

	var result []Stat
	for _, change := range changes {
		s, err := c.fileStatsFromChange(change)
		if err != nil {
			if err == errIgnored {
				continue
			}
			return nil, err
		}

		result = append(result, s)
	}

	return result, nil
}

func (c CLI) fileStatsFromChange(change *object.Change) (Stat, error) {
	action, err := change.Action()
	if err != nil {
		return Stat{}, err
	}

	var kind string
	var path string
	switch action {
	case merkletrie.Delete:
		kind = "delete"
		path = change.From.Name
	case merkletrie.Insert:
		kind = "insert"
		path = change.To.Name
	case merkletrie.Modify:
		kind = "modify"
		path = change.To.Name
	default:
		kind = "unknown"
	}

	return Stat{
		Kind: kind,
		Path: path,
	}, nil
}

var errIgnored = errors.New("ignored file")

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
