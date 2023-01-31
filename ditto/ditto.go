package ditto

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/b4b4r07/changed-objects/git"
	"github.com/bmatcuk/doublestar/v4"
)

type File struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Kind      git.Kind  `json:"kind"`
	ParentDir ParentDir `json:"parent_dir"`
}

type Files []File

type ParentDir struct {
	Path  string `json:"path"`
	Exist bool   `json:"exist"`
}

type Dir struct {
	Path  string `json:"path"`
	Files Files  `json:"files"`
}

type Dirs []Dir

type Option struct {
	DirExist      bool
	DirNotExist   bool
	DefaultBranch string
	MergeBase     string

	OnlyDir bool
}

type client struct {
	args    []string
	opt     Option
	changes []git.Change
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
	return client{
		args:    args,
		opt:     opt,
		changes: changes,
	}, nil
}

// func (fs *Files) Filter(f func(File) bool) Files {
// 	files := make(Files, 0)
// 	for _, file := range *fs {
// 		if f(file) {
// 			files = append(files, file)
// 		}
// 	}
// 	return files
// }
//
// func (ds *Dirs) Filter(f func(Dir) bool) Dirs {
// 	dirs := make(Dirs, 0)
// 	for _, dir := range *ds {
// 		if f(dir) {
// 			dirs = append(dirs, dir)
// 		}
// 	}
// 	return dirs
// }

func (c client) GetFiles() (Files, error) {
	var files Files

	for _, change := range c.changes {
		if len(c.args) > 0 {
			arg := c.args[0]
			matched, _ := doublestar.Match(filepath.Join(arg, "**"), change.Path)
			if !matched {
				log.Printf("[DEBUG] GetDirs: %s is not matched in %s\n", change.Path, arg)
				continue
			}
		}
		files = append(files, getFile(change))
	}

	// if opt.DirExist {
	// 	stats = stats.Filter(func(stat Stat) bool {
	// 		return stat.DirExist
	// 	})
	// }
	//
	// if opt.DirNotExist {
	// 	stats = stats.Filter(func(stat Stat) bool {
	// 		return !stat.DirExist
	// 	})
	// }
	//
	// // OnlyDir
	// if opt.OnlyDir {
	// 	stats = stats.Dirs()
	// }

	return files, nil
}

func getFile(change git.Change) File {
	return File{
		Name: filepath.Base(change.Path),
		Path: change.Path,
		Kind: change.Kind,
		ParentDir: ParentDir{
			Path: change.Dir,
			Exist: func() bool {
				_, err := os.Stat(change.Dir)
				return err == nil
			}(),
		},
	}
}

func (c client) GetDirs() ([]Dir, error) {
	matrix := make(map[string]Dir)

	for _, change := range c.changes {
		path := change.Dir
		if len(c.args) > 0 {
			arg := c.args[0]
			length := len(strings.Split(arg, "/"))
			matched, _ := doublestar.Match(filepath.Join(arg, "**"), change.Path)
			if !matched {
				log.Printf("[DEBUG] GetDirs: %s is not matched in %s\n", change.Path, arg)
				continue
			}
			path = strings.Join(strings.Split(change.Path, "/")[0:length], "/")
			log.Printf("[DEBUG] GetDirs: chunk path %s\n", path)
		}
		dir, ok := matrix[path]
		if ok {
			dir.Files = append(dir.Files, getFile(change))
		} else {
			dir = Dir{
				Path:  path,
				Files: Files{getFile(change)},
			}
		}
		matrix[path] = dir
	}

	var dirs Dirs
	for _, dir := range matrix {
		dirs = append(dirs, dir)
	}

	// if opt.DirExist {
	// 	stats = stats.Filter(func(stat Stat) bool {
	// 		return stat.DirExist
	// 	})
	// }
	//
	// if opt.DirNotExist {
	// 	stats = stats.Filter(func(stat Stat) bool {
	// 		return !stat.DirExist
	// 	})
	// }
	//
	// // OnlyDir
	// if opt.OnlyDir {
	// 	stats = stats.Dirs()
	// }

	return dirs, nil
}
