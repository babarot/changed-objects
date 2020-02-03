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
	Filters []string `long:"filter" description:"Filter the kind of changed objects (added/deleted/modified)" default:"all"`
	Dirname bool     `long:"dirname" description:"Return changed objects with their directory name"`
	Output  string   `long:"output" short:"o" description:"Format to output the result" default:""`

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
		fmt.Fprintf(c.Stdout, "%s (%s)\n", Version, Revision)
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
	for _, filter := range c.Option.Filters {
		switch filter {
		case "all":
			ss = stats
			break
		case "added":
			ss = append(ss, stats.Filter(func(stat Stat) bool {
				return stat.Kind == Addition
			})...)
		case "deleted":
			ss = append(ss, stats.Filter(func(stat Stat) bool {
				return stat.Kind == Deletion
			})...)
		case "modified":
			ss = append(ss, stats.Filter(func(stat Stat) bool {
				return stat.Kind == Modification
			})...)
		case "":
			return fmt.Errorf("requires a filter at least one")
		default:
			return fmt.Errorf("%s: invalid filter (added,deleted,modified can be allowed)", filter)
		}
	}
	stats = ss

	if c.Option.Dirname {
		stats = stats.Map(func(stat Stat) Stat {
			return Stat{
				Kind: stat.Kind,
				Path: filepath.Dir(stat.Path),
			}
		})
		stats = stats.Unique()
	}

	switch c.Option.Output {
	case "json":
		result := Result{Stats: stats}
		result.Print(c.Stdout)
	default:
		for _, stat := range stats {
			fmt.Fprintln(c.Stdout, stat.Path)
		}
	}

	return nil
}

func (c CLI) getStats(from, to *object.Commit) (Stats, error) {
	src, err := to.Tree()
	if err != nil {
		return Stats{}, err
	}

	dst, err := from.Tree()
	if err != nil {
		return Stats{}, err
	}

	changes, err := object.DiffTree(dst, src)
	if err != nil {
		return Stats{}, err
	}

	var stats []Stat
	for _, change := range changes {
		stat, err := c.fileStatsFromChange(change)
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}
