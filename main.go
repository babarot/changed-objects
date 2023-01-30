package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	clilog "github.com/b4b4r07/go-cli-log"
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

type Option struct {
	Filters       []string `long:"filter" description:"Filter the kind of changed objects" default:"all" choice:"added" choice:"modified" choice:"deleted" choice:"all"`
	Dirname       bool     `long:"dirname" description:"Return changed objects with their directory name"`
	DirExist      bool     `long:"dir-exist" description:"Return changed objects if parent dir exists"`
	DirNotExist   bool     `long:"dir-not-exist" description:"Return changed objects if parent dir does not exist"`
	Output        string   `long:"output" short:"o" description:"Format to output the result" default:"" choice:"json"`
	DefaultBranch string   `long:"default-branch" description:"Specify default branch" default:"main"`
	MergeBase     string   `long:"merge-base" description:"Specify merge-base revision"`

	Version bool `short:"v" long:"version" description:"Show version"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	clilog.Env = "LOG"
	clilog.SetOutput()
	defer log.Printf("[INFO] finish main function")

	log.Printf("[INFO] Version: %s (%s)", Version, Revision)
	log.Printf("[INFO] Args: %#v", args)

	var opt Option
	args, err := flags.ParseArgs(&opt, args)
	if err != nil {
		return err
	}

	switch {
	case opt.Version:
		fmt.Fprintf(os.Stdout, "%s (%s)\n", Version, Revision)
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
	log.Printf("[INFO] git repo: %s", abs)

	repo, err := git.PlainOpen(abs)
	if err != nil {
		return fmt.Errorf("cannot open repository: %w", err)
	}

	branch, err := GetCurrentBranchFromRepository(repo)
	if err != nil {
		return err
	}
	log.Printf("[TRACE] Getting current branch: %s", branch)

	var base *object.Commit

	switch branch {
	case opt.DefaultBranch:
		log.Printf("[DEBUG] Getting previous HEAD commit")
		prev, err := previousCommit(repo)
		if err != nil {
			return err
		}
		base = prev
	default:
		log.Printf("[DEBUG] Getting remote commit")
		remote, err := remoteCommit("origin/"+opt.DefaultBranch, repo)
		if err != nil {
			return err
		}
		base = remote
	}

	if opt.MergeBase != "" {
		log.Printf("[DEBUG] Comparing with merge-base")
		h, err := repo.Head()
		if err != nil {
			return err
		}
		currentBranch := h.Name().Short()
		base, err = mergeBase(opt.MergeBase, currentBranch, repo)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Getting current commit")
	current, err := currentCommit(repo)
	if err != nil {
		return err
	}

	stats, err := getStats(base, current, repo)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Option filters: %#v", opt.Filters)
	var ss Stats
	for _, filter := range opt.Filters {
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

	if opt.Dirname {
		stats = stats.Map(func(stat Stat) Stat {
			stat.Path = stat.Dir
			return stat
		})
	}

	if opt.DirExist {
		stats = stats.Filter(func(stat Stat) bool {
			return stat.DirExist
		})
	}

	if opt.DirNotExist {
		stats = stats.Filter(func(stat Stat) bool {
			return !stat.DirExist
		})
	}

	stats = stats.Unique()

	switch opt.Output {
	case "json":
		r := struct {
			Repo  string `json:"repo"`
			Stats Stats  `json:"stats"`
		}{
			Repo:  abs,
			Stats: stats,
		}
		return json.NewEncoder(os.Stdout).Encode(&r)
	case "":
		for _, stat := range stats {
			fmt.Fprintln(os.Stdout, stat.Path)
		}
	default:
		return fmt.Errorf("%s: invalid output format", opt.Output)
	}

	return nil
}

func getStats(from, to *object.Commit, repo *git.Repository) (Stats, error) {
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
		stat, err := fileStatsFromChange(change, repo)
		if err != nil {
			continue
		}
		log.Printf("[DEBUG] stat: %#v", stat)
		stats = append(stats, stat)
	}

	return stats, nil
}

func currentCommit(repo *git.Repository) (*object.Commit, error) {
	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] %s: get commit", ref.Name().String())
	return repo.CommitObject(ref.Hash())
}

func previousCommit(repo *git.Repository) (*object.Commit, error) {
	hash, err := repo.ResolveRevision("HEAD^")
	if err != nil {
		return nil, err
	}

	return repo.CommitObject(*hash)
}

func remoteCommit(name string, repo *git.Repository) (*object.Commit, error) {
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}

	var cmt *object.Commit
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().String() == fmt.Sprintf("refs/remotes/%s", name) {
			commit, err := repo.CommitObject(ref.Hash())
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] %s: get commit", ref.Name().String())
			cmt = commit
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cmt, nil
}

func masterCommit(name string, repo *git.Repository) (*object.Commit, error) {
	branches, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	var cmt *object.Commit
	err = branches.ForEach(func(branch *plumbing.Reference) error {
		if branch.Name().String() == fmt.Sprintf("refs/heads/%s", name) {
			commit, err := repo.CommitObject(branch.Hash())
			if err != nil {
				return err
			}
			log.Printf("[DEBUG] %s: get commit", branch.Name().String())
			cmt = commit
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cmt, nil
}

func fileStatsFromChange(change *object.Change, repo *git.Repository) (Stat, error) {
	action, err := change.Action()
	if err != nil {
		return Stat{}, err
	}

	var kind Kind
	var path string
	switch action {
	case merkletrie.Delete:
		kind = Deletion
		path = change.From.Name
	case merkletrie.Insert:
		kind = Addition
		path = change.To.Name
	case merkletrie.Modify:
		kind = Modification
		path = change.To.Name
	default:
		kind = Unknown
	}

	return Stat{
		Kind: kind,
		Path: path,
		File: path,
		Dir:  filepath.Dir(path),
		DirExist: func() bool {
			_, err := os.Stat(filepath.Dir(path))
			return err == nil
		}(),
	}, nil
}

// https://github.com/go-git/go-git/blob/master/_examples/merge_base/main.go
func mergeBase(baseRev, commitRev string, repo *git.Repository) (*object.Commit, error) {
	log.Printf("[DEBUG] baseRev: %s, commitRev: %s", baseRev, commitRev)

	// Get the hashes of the passed revisions
	var hashes []*plumbing.Hash
	for _, rev := range []string{baseRev, commitRev} {
		hash, err := repo.ResolveRevision(plumbing.Revision(rev))
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	// Get the commits identified by the passed hashes
	var commits []*object.Commit
	for _, hash := range hashes {
		commit, err := repo.CommitObject(*hash)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}

	res, err := commits[0].MergeBase(commits[1])
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, errors.New("failed to get merge-base")
	}

	return res[0], nil
}

// https://github.com/src-d/go-git/issues/1030
func GetCurrentBranchFromRepository(repo *git.Repository) (string, error) {
	branchRefs, err := repo.Branches()
	if err != nil {
		return "", err
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	var currentBranchName string
	err = branchRefs.ForEach(func(branchRef *plumbing.Reference) error {
		if branchRef.Hash() == headRef.Hash() {
			currentBranchName = branchRef.Name().Short()
			return nil
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return currentBranchName, nil
}

type Kind int

const (
	Addition Kind = iota
	Deletion
	Modification
	Unknown
)

func (k Kind) String() string {
	switch k {
	case Addition:
		return "insert"
	case Deletion:
		return "delete"
	case Modification:
		return "modify"
	default:
		return "unknown"
	}
}

func (k Kind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// Stat represents the stats for a file in a commit.
type Stat struct {
	Kind     Kind   `json:"kind"`
	Path     string `json:"path"`
	File     string `json:"file"`
	Dir      string `json:"dir"`
	DirExist bool   `json:"dir-exist"`
	Output   string `json:"-"`
}

type Stats []Stat

func (ss *Stats) Filter(f func(Stat) bool) Stats {
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if f(stat) {
			stats = append(stats, stat)
		}
	}
	return stats
}

func (ss *Stats) Map(f func(Stat) Stat) Stats {
	stats := make([]Stat, len(*ss))
	for i, stat := range *ss {
		stats[i] = f(stat)
	}
	return stats
}

func (ss *Stats) Unique() Stats {
	m := make(map[string]bool)
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if !m[stat.Path] {
			m[stat.Path] = true
			stats = append(stats, stat)
		}
	}
	return stats
}
