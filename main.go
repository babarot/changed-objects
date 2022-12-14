package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	clilog "github.com/b4b4r07/go-cli-log"
	"github.com/bmatcuk/doublestar/v4"
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
	Filters       []string `long:"filter" description:"Filter the kind of changed objects" default:"all" choice:"added" choice:"modified" choice:"deleted" choice:"all"`
	Dirname       bool     `long:"dirname" description:"Return changed objects with their directory name"`
	DirExist      bool     `long:"dir-exist" description:"Return changed objects if parent dir exists"`
	DirNotExist   bool     `long:"dir-not-exist" description:"Return changed objects if parent dir does not exist"`
	Output        string   `long:"output" short:"o" description:"Format to output the result" default:"" choice:"json"`
	DefaultBranch string   `long:"default-branch" description:"Specify default branch" default:"main"`
	MergeBase     string   `long:"merge-base" description:"Specify merge-base revision"`

	Ignores []string `long:"ignore" description:"Ignore string pattern"`

	Version bool `short:"v" long:"version" description:"Show version"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	clilog.Env = "LOG"
	clilog.SetOutput()
	defer log.Printf("[INFO] finish main function")

	log.Printf("[INFO] Version: %s (%s)", Version, Revision)
	log.Printf("[INFO] Args: %#v", args)

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

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(wd)
	if err != nil {
		return err
	}
	repo := abs
	log.Printf("[INFO] git repo: %s", repo)

	r, err := git.PlainOpen(repo)
	if err != nil {
		return fmt.Errorf("cannot open repository: %w", err)
	}
	c.Repo = r

	branch, err := c.GetCurrentBranchFromRepository()
	if err != nil {
		return err
	}
	log.Printf("[TRACE] Getting current branch: %s", branch)

	var base *object.Commit

	switch branch {
	case c.Option.DefaultBranch:
		log.Printf("[DEBUG] Getting previous HEAD commit")
		prev, err := c.previousCommit()
		if err != nil {
			return err
		}
		base = prev
	default:
		log.Printf("[DEBUG] Getting remote commit")
		remote, err := c.remoteCommit("origin/" + c.Option.DefaultBranch)
		if err != nil {
			return err
		}
		base = remote
	}

	if c.Option.MergeBase != "" {
		log.Printf("[DEBUG] Comparing with merge-base")
		h, err := c.Repo.Head()
		if err != nil {
			return err
		}
		currentBranch := h.Name().Short()
		base, err = c.mergeBase(c.Option.MergeBase, currentBranch)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Getting current commit")
	current, err := c.currentCommit()
	if err != nil {
		return err
	}

	stats, err := c.getStats(base, current)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Option filters: %#v", c.Option.Filters)
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
		}
	}
	stats = ss

	if len(args) > 0 {
		var ss Stats
		log.Printf("[TRACE] Filtering with args")
		for _, arg := range args {
			ss = append(ss, stats.Filter(func(stat Stat) bool {
				log.Printf("[TRACE] Filtering with stat %q, file %q", stat.Path, arg)
				return strings.Index(stat.Path, arg) == 0
			})...)
		}
		stats = ss
	}

	if c.Option.Dirname {
		stats = stats.Map(func(stat Stat) Stat {
			stat.Path = stat.Dir
			return stat
		})
	}

	if c.Option.DirExist {
		stats = stats.Filter(func(stat Stat) bool {
			return stat.DirExist
		})
	}

	if c.Option.DirNotExist {
		stats = stats.Filter(func(stat Stat) bool {
			return !stat.DirExist
		})
	}

	for _, ignore := range c.Option.Ignores {
		stats = stats.Filter(func(stat Stat) bool {
			match, err := doublestar.Match(ignore, stat.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
				return false
			}
			return !match
		})
	}

	stats = stats.Unique()

	switch c.Option.Output {
	case "json":
		r := struct {
			Repo  string `json:"repo"`
			Stats Stats  `json:"stats"`
		}{
			Repo:  repo,
			Stats: stats,
		}
		return json.NewEncoder(c.Stdout).Encode(&r)
	case "":
		for _, stat := range stats {
			fmt.Fprintln(c.Stdout, stat.Path)
		}
	default:
		return fmt.Errorf("%s: invalid output format", c.Option.Output)
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

	log.Printf("[DEBUG] a number of changes: %d", len(changes))

	var stats []Stat
	for _, change := range changes {
		stat, err := c.fileStatsFromChange(change)
		if err != nil {
			continue
		}
		log.Printf("[DEBUG] stat: %#v", stat)
		stats = append(stats, stat)
	}

	return stats, nil
}
