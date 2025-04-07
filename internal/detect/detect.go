package detect

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/babarot/changed-objects/internal/git"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/k0kubun/pp/v3"
	"github.com/samber/lo"
)

type client struct {
	args    []string
	opt     Option
	changes []git.Change
	pp      *pp.PrettyPrinter
}

type Option struct {
	DefaultBranch string
	MergeBase     string
	Types         []string
	Ignores       []string
	GroupBy       []string
	DirExist      string
}

func New(path string, args []string, opt Option) (client, error) {
	changes, err := git.Open(git.Config{
		Path:          path,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
	})
	if err != nil {
		return client{}, err
	}

	printer := pp.New()
	printer.SetColoringEnabled(false)
	printer.SetExportedOnly(true)
	return client{
		args:    args,
		opt:     opt,
		changes: changes,
		pp:      printer,
	}, nil
}

func (c client) Run() (Diff, error) {
	changes := c.changes

	for _, arg := range c.args {
		// filter by given dir names
		changes = lo.Filter(changes, func(change git.Change, _ int) bool {
			return strings.Index(filepath.Dir(change.Path), arg) == 0
		})
	}

	for _, ignore := range c.opt.Ignores {
		// filter out by given patterns
		changes = lo.Filter(changes, func(change git.Change, _ int) bool {
			match, err := doublestar.Match(ignore, filepath.Dir(change.Path))
			if err != nil {
				return false
			}
			return !match
		})
	}

	if len(c.opt.Types) > 0 {
		// filter by change type
		filtered := []git.Change{}
		for _, ty := range c.opt.Types {
			filtered = append(filtered, lo.Filter(changes, func(change git.Change, _ int) bool {
				switch ty {
				case "added":
					return change.Type == git.Addition
				case "deleted":
					return change.Type == git.Deletion
				case "modified":
					return change.Type == git.Modification
				}
				return false
			})...)
		}
		changes = filtered
	}

	// filter by the existence of parent dir
	changes = lo.Filter(changes, func(change git.Change, _ int) bool {
		_, err := os.Stat(filepath.Dir(change.Path))
		exist := err == nil
		switch c.opt.DirExist {
		case "true":
			return exist
		case "false":
			return !exist
		default:
			return true
		}
	})

	return Diff{
		Files: c.getFiles(changes),
		Dirs:  c.getDirs(changes),
	}, nil
}

func (c client) getFiles(changes []git.Change) []File {
	var files []File

	for _, change := range changes {
		files = append(files, getFile(change))
	}
	return files
}

func (c client) getDirs(changes []git.Change) []Dir {
	matrix := make(map[string]Dir)
	for path, changes := range findDirWithPatterns(changes, c.opt.GroupBy) {
		for _, change := range changes {
			dir, ok := matrix[path]
			if ok {
				log.Printf("[TRACE] getDirs: updated %q", path)
				dir.Files = append(dir.Files, getFile(change))
			} else {
				log.Printf("[TRACE] getDirs: created %q", path)
				dir = Dir{
					Path: path,
					Exist: func() bool {
						_, err := os.Stat(path)
						return err == nil
					}(),
					Files: []File{getFile(change)},
				}
			}
			matrix[path] = dir
		}
	}

	var dirs []Dir
	for _, dir := range matrix {
		dirs = append(dirs, dir)
	}
	return dirs
}

func getSteps(path string) []string {
	var steps []string
	step := path
	for {
		steps = append(steps, step)
		step = filepath.Dir(step)
		if step == "." || step == "/" {
			break
		}
	}
	return steps
}

func findDirWithPatterns(changes []git.Change, patterns []string) map[string][]git.Change {
	found := make(map[string][]git.Change)

	if len(patterns) == 0 {
		// If no patterns are specified, use the direct parent directory of each file
		for _, change := range changes {
			parentDir := filepath.Dir(change.Path)
			found[parentDir] = append(found[parentDir], change)
		}
		return found
	}

	// If patterns are specified, use the minimum match
	min := true

	for _, change := range changes {
		steps := getSteps(filepath.Dir(change.Path))
		var dirs []string
		for _, pattern := range patterns {
			dirs = append(dirs, lo.FilterMap(steps, func(step string, _ int) (string, bool) {
				matched, _ := doublestar.Match(pattern, step)
				return step, matched
			})...)
		}
		if len(dirs) == 0 {
			continue
		}
		var dir string
		if min {
			dir = lo.MinBy(dirs, func(item string, dir string) bool {
				return len(strings.Split(item, "/")) < len(strings.Split(dir, "/"))
			})
		} else {
			dir = lo.MaxBy(dirs, func(item string, dir string) bool {
				return len(strings.Split(item, "/")) > len(strings.Split(dir, "/"))
			})
		}
		found[dir] = append(found[dir], change)
	}

	return found
}
