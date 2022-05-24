package main

import (
	"errors"
	"fmt"
	"log"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

func (c CLI) currentCommit() (*object.Commit, error) {
	ref, err := c.Repo.Head()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] %s: get commit", ref.Name().String())
	return c.Repo.CommitObject(ref.Hash())
}

func (c CLI) previousCommit() (*object.Commit, error) {
	hash, err := c.Repo.ResolveRevision("HEAD^")
	if err != nil {
		return nil, err
	}

	return c.Repo.CommitObject(*hash)
}

func (c CLI) remoteCommit(name string) (*object.Commit, error) {
	refs, err := c.Repo.References()
	if err != nil {
		return nil, err
	}

	var cmt *object.Commit
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().String() == fmt.Sprintf("refs/remotes/%s", name) {
			commit, err := c.Repo.CommitObject(ref.Hash())
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

func (c CLI) masterCommit(name string) (*object.Commit, error) {
	branches, err := c.Repo.Branches()
	if err != nil {
		return nil, err
	}

	var cmt *object.Commit
	err = branches.ForEach(func(branch *plumbing.Reference) error {
		if branch.Name().String() == fmt.Sprintf("refs/heads/%s", name) {
			commit, err := c.Repo.CommitObject(branch.Hash())
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

func (c CLI) fileStatsFromChange(change *object.Change) (Stat, error) {
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
	}, nil
}

// https://github.com/go-git/go-git/blob/master/_examples/merge_base/main.go
func (c CLI) mergeBase(baseRev, commitRev string) (*object.Commit, error) {
	log.Printf("[DEBUG] baseRev: %s, commitRev: %s", baseRev, commitRev)
	repo := c.Repo

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
func (c CLI) GetCurrentBranchFromRepository() (string, error) {
	branchRefs, err := c.Repo.Branches()
	if err != nil {
		return "", err
	}

	headRef, err := c.Repo.Head()
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
