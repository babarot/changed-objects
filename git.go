package main

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

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
