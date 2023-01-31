package git

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

type Config struct {
	repo *git.Repository

	Path          string
	DefaultBranch string
	MergeBase     string
}

type Change struct {
	Path string
	Dir  string
	Kind Kind
}

func Open(cfg Config) ([]Change, error) {
	repo, err := git.PlainOpen(cfg.Path)
	if err != nil {
		return []Change{}, fmt.Errorf("cannot open repository: %w", err)
	}
	cfg.repo = repo

	branch, err := cfg.getCurrentBranch()
	if err != nil {
		return []Change{}, err
	}
	log.Printf("[TRACE] Getting current branch: %s", branch)

	var base *object.Commit

	switch branch {
	case cfg.DefaultBranch:
		log.Printf("[DEBUG] Getting previous HEAD commit")
		prev, err := cfg.previousCommit()
		if err != nil {
			return []Change{}, err
		}
		base = prev
	default:
		log.Printf("[DEBUG] Getting remote commit")
		remote, err := cfg.remoteCommit("origin/" + cfg.DefaultBranch)
		if err != nil {
			return []Change{}, err
		}
		base = remote
	}

	if len(cfg.MergeBase) > 0 {
		log.Printf("[DEBUG] Comparing with merge-base")
		h, err := cfg.repo.Head()
		if err != nil {
			return []Change{}, err
		}
		currentBranch := h.Name().Short()
		base, err = cfg.mergeBaseCommit(cfg.MergeBase, currentBranch)
		if err != nil {
			return []Change{}, err
		}
	}

	log.Printf("[DEBUG] Getting current commit")
	current, err := cfg.currentCommit()
	if err != nil {
		return []Change{}, err
	}

	return cfg.getChanges(base, current)
}

// https://github.com/src-d/go-git/issues/1030
func (c Config) getCurrentBranch() (string, error) {
	branchRefs, err := c.repo.Branches()
	if err != nil {
		return "", err
	}

	headRef, err := c.repo.Head()
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

func (c Config) currentCommit() (*object.Commit, error) {
	ref, err := c.repo.Head()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] %s: get commit", ref.Name().String())
	return c.repo.CommitObject(ref.Hash())
}

func (c Config) previousCommit() (*object.Commit, error) {
	hash, err := c.repo.ResolveRevision("HEAD^")
	if err != nil {
		return nil, err
	}

	return c.repo.CommitObject(*hash)
}

func (c Config) remoteCommit(name string) (*object.Commit, error) {
	refs, err := c.repo.References()
	if err != nil {
		return nil, err
	}

	var cmt *object.Commit
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().String() == fmt.Sprintf("refs/remotes/%s", name) {
			commit, err := c.repo.CommitObject(ref.Hash())
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

// https://github.com/go-git/go-git/blob/master/_examples/merge_base/main.go
func (c Config) mergeBaseCommit(baseRev, commitRev string) (*object.Commit, error) {
	log.Printf("[DEBUG] baseRev: %s, commitRev: %s", baseRev, commitRev)

	// Get the hashes of the passed revisions
	var hashes []*plumbing.Hash
	for _, rev := range []string{baseRev, commitRev} {
		hash, err := c.repo.ResolveRevision(plumbing.Revision(rev))
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	// Get the commits identified by the passed hashes
	var commits []*object.Commit
	for _, hash := range hashes {
		commit, err := c.repo.CommitObject(*hash)
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

func (c Config) getChanges(from, to *object.Commit) ([]Change, error) {
	src, err := to.Tree()
	if err != nil {
		return []Change{}, err
	}

	dst, err := from.Tree()
	if err != nil {
		return []Change{}, err
	}

	changes, err := object.DiffTree(dst, src)
	if err != nil {
		return []Change{}, err
	}

	log.Printf("[DEBUG] a number of changes: %d", len(changes))

	var cs []Change
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			return []Change{}, err
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
		cs = append(cs, Change{
			Path: path,
			Dir:  filepath.Dir(path),
			Kind: kind,
		})
	}

	return cs, nil
}
