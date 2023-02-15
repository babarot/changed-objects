package detect

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/b4b4r07/changed-objects/internal/git"
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
	files, err := c.getFiles()
	if err != nil {
		return Diff{}, err
	}

	dirs, err := c.getDirs()
	if err != nil {
		return Diff{}, err
	}

	// if len(c.opt.Types) > 0 {
	// 	filtered := Files{}
	// 	filtered = append(filtered, lo.Filter[File](files, func(file File, _ int) bool {
	// 		switch file.Type.String() {
	// 		case "added":
	// 			return file.Type == git.Addition
	// 		case "deleted":
	// 			return file.Type == git.Deletion
	// 		case "modified":
	// 			return file.Type == git.Modification
	// 		}
	// 		return false
	// 	})...)
	// 	files = filtered
	// }

	if len(c.opt.Types) > 0 {
		filtered := Files{}
		for _, ty := range c.opt.Types {
			filtered = append(filtered, lo.Filter[File](files, func(file File, _ int) bool {
				switch ty {
				case "added":
					return file.Type == git.Addition
				case "deleted":
					return file.Type == git.Deletion
				case "modified":
					return file.Type == git.Modification
				}
				return false
			})...)
		}
		files = filtered

		tmpDirs := Dirs{}
		for _, dir := range dirs {
			files := Files{}
			for _, ty := range c.opt.Types {
				files = append(files, lo.Filter[File](dir.Files, func(file File, _ int) bool {
					switch ty {
					case "added":
						return file.Type == git.Addition
					case "deleted":
						return file.Type == git.Deletion
					case "modified":
						return file.Type == git.Modification
					}
					return false
				})...)
			}
			// files := Files{}
			// for _, ty := range c.opt.Types {
			// 	switch ty {
			// 	case "added":
			// 		files = append(files, dir.Files.filter(func(file File) bool {
			// 			return file.Type == git.Addition
			// 		})...)
			// 	case "deleted":
			// 		files = append(files, dir.Files.filter(func(file File) bool {
			// 			return file.Type == git.Deletion
			// 		})...)
			// 	case "modified":
			// 		files = append(files, dir.Files.filter(func(file File) bool {
			// 			return file.Type == git.Modification
			// 		})...)
			// 	}
			// }
			if len(files) > 0 {
				dir.Files = files
				tmpDirs = append(tmpDirs, dir)
			}
		}
		dirs = tmpDirs
	}

	files = files.filter(func(file File) bool {
		switch c.opt.DirExist {
		case "true":
			return file.ParentDir.Exist
		case "false":
			return !file.ParentDir.Exist
		default:
			return true
		}
	})

	dirs = dirs.filter(func(dir Dir) bool {
		switch c.opt.DirExist {
		case "true":
			return dir.Exist
		case "false":
			return !dir.Exist
		default:
			return true
		}
	})

	return Diff{
		Files: files,
		Dirs:  dirs,
	}, nil
}

func (c client) getFiles() (Files, error) {
	var files Files

	for _, change := range c.changes {
		files = append(files, getFile(change))
	}

	for _, arg := range c.args {
		files = files.filter(func(file File) bool {
			return strings.Index(file.Path, arg) == 0
		})
	}

	for _, ignore := range c.opt.Ignores {
		files = files.filter(func(file File) bool {
			match, err := doublestar.Match(ignore, file.Path)
			if err != nil {
				return false
			}
			return !match
		})
	}

	return files, nil
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

// find directory (and git changes which it has) matched with given patterns.
//
// patterns:
//   [
//     "kubernetes/**/{base,overlays}",
//     "kubernetes/**/{development,production}",
//   ]
// git changes:
//   [
//     "kubernetes/playground-cluster/cluster-resources/development/Namespace/service-A.yaml"
//     "kubernetes/playground-cluster/namespaces/service-A/development/CronJob/test.yaml"
//     "kubernetes/playground-cluster/namespaces/service-B/base/kustomization.yaml"
//     "kubernetes/playground-cluster/namespaces/service-B/overlays/development/kustomization.yaml"
//     "kubernetes/playground-cluster/namespaces/service-B/overlays/production/kustomization.yaml"
//   ]
// into this result:
//   map[string][]git.Change{
//     "kubernetes/playground-cluster/cluster-resources/development": [
//        git.Change{"Path": "kubernetes/playground-cluster/cluster-resources/development/Namespace/service-A.yaml"},
//     ]
//     "kubernetes/playground-cluster/namespaces/service-A/development": [
//        git.Change{"Path": "kubernetes/playground-cluster/namespaces/service-A/development/CronJob/test.yaml"},
//     ]
//     "kubernetes/playground-cluster/namespaces/service-B/base": [
//        git.Change{"Path": "kubernetes/playground-cluster/namespaces/service-B/base/kustomization.yaml"},
//     ]
//     "kubernetes/playground-cluster/namespaces/service-B/overlays": [
//        git.Change{"Path": "kubernetes/playground-cluster/namespaces/service-B/overlays/development/kustomization.yaml"},
//        git.Change{"Path": "kubernetes/playground-cluster/namespaces/service-B/overlays/production/kustomization.yaml"},
//     ]
//   }
func findDirWithPatterns(changes []git.Change, patterns []string) map[string][]git.Change {
	found := make(map[string][]git.Change)

	if len(patterns) == 0 {
		for _, change := range changes {
			// if no given patterns, find files located in parent dir.
			patterns = append(patterns, filepath.Dir(change.Path))
		}
	}

	for _, change := range changes {
		steps := getSteps(filepath.Dir(change.Path))
		var dirs []string
		for _, pattern := range patterns {
			dirs = append(dirs, lo.FilterMap[string, string](steps, func(step string, _ int) (string, bool) {
				matched, _ := doublestar.Match(pattern, step)
				return step, matched
			})...)
		}
		if len(dirs) == 0 {
			continue
		}
		dir := lo.MinBy(dirs, func(item string, dir string) bool {
			return len(strings.Split(item, "/")) < len(strings.Split(dir, "/"))
		})
		found[dir] = append(found[dir], change)
	}

	return found
}

func (c client) getDirs() (Dirs, error) {
	matrix := make(map[string]Dir)

	for path, changes := range findDirWithPatterns(c.changes, c.opt.GroupBy) {
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
					Files: Files{getFile(change)},
				}
			}
			matrix[path] = dir
		}
	}

	var dirs Dirs
	for _, dir := range matrix {
		log.Printf("[TRACE] getDirs: convert dirs matrix to slice: %q", dir.Path)
		dirs = append(dirs, dir)
	}

	for _, arg := range c.args {
		dirs = dirs.filter(func(dir Dir) bool {
			return strings.Index(dir.Path, arg) == 0
		})
	}

	for _, ignore := range c.opt.Ignores {
		dirs = dirs.filter(func(dir Dir) bool {
			match, err := doublestar.Match(ignore, dir.Path)
			if err != nil {
				return false
			}
			return !match
		})
	}

	return dirs, nil
}
