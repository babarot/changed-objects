package ditto

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/b4b4r07/changed-objects/internal/git"
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
	DefaultBranch string
	MergeBase     string
	Filters       []string
	Ignores       []string
	GroupBy       string
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

type Object interface {
	GetPath() string
}

func (f File) GetPath() string {
	return f.Path
}

func (d Dir) GetPath() string {
	return d.Path
}

func (fs *Files) Filter(f func(File) bool) Files {
	files := make(Files, 0)
	for _, file := range *fs {
		if f(file) {
			files = append(files, file)
		}
	}
	return files
}

func (ds *Dirs) Filter(f func(Dir) bool) Dirs {
	dirs := make(Dirs, 0)
	for _, dir := range *ds {
		if f(dir) {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

type Result struct {
	Files Files `json:"files"`
	Dirs  Dirs  `json:"dirs"`
}

func (c client) Get() (Result, error) {
	files, err := c.GetFiles()
	if err != nil {
		return Result{}, err
	}
	dirs, err := c.GetDirs()
	if err != nil {
		return Result{}, err
	}

	if len(c.opt.Filters) > 0 {
		tmpFiles := Files{}
		tmpDirs := Dirs{}
		for _, filter := range c.opt.Filters {
			switch filter {
			case "added":
				tmpFiles = append(tmpFiles, files.Filter(func(file File) bool {
					return file.Kind == git.Addition
				})...)
				tmpDirs = append(tmpDirs, dirs.Filter(func(dir Dir) bool {
					files := dir.Files.Filter(func(file File) bool {
						return file.Kind == git.Addition
					})
					dir.Files = files
					return len(files) > 0
				})...)
			case "deleted":
				tmpFiles = append(tmpFiles, files.Filter(func(file File) bool {
					return file.Kind == git.Deletion
				})...)
				tmpDirs = append(tmpDirs, dirs.Filter(func(dir Dir) bool {
					files := dir.Files.Filter(func(file File) bool {
						return file.Kind == git.Deletion
					})
					dir.Files = files
					return len(files) > 0
				})...)
			case "modified":
				tmpFiles = append(tmpFiles, files.Filter(func(file File) bool {
					return file.Kind == git.Modification
				})...)
				tmpDirs = append(tmpDirs, dirs.Filter(func(dir Dir) bool {
					files := dir.Files.Filter(func(file File) bool {
						return file.Kind == git.Modification
					})
					dir.Files = files
					return len(files) > 0
				})...)
			}
		}
		files = tmpFiles
		dirs = tmpDirs
	}

	return Result{
		Files: files,
		Dirs:  dirs,
	}, nil
}

func (c client) GetFiles() (Files, error) {
	var files Files

	for _, change := range c.changes {
		if len(c.opt.GroupBy) > 0 {
			matched, _ := doublestar.Match(filepath.Join(c.opt.GroupBy, "**"), change.Path)
			if !matched {
				log.Printf("[DEBUG] GetFiles: %s is not matched in %s\n", change.Path, c.opt.GroupBy)
				continue
			}
		}
		files = append(files, getFile(change))
	}

	for _, arg := range c.args {
		files = files.Filter(func(file File) bool {
			return strings.Index(file.Path, arg) == 0
		})
	}

	for _, ignore := range c.opt.Ignores {
		files = files.Filter(func(file File) bool {
			match, err := doublestar.Match(ignore, file.Path)
			if err != nil {
				return false
			}
			return !match
		})
	}

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

func (c client) GetDirs() (Dirs, error) {
	matrix := make(map[string]Dir)

	for _, change := range c.changes {
		path := change.Dir
		if len(c.opt.GroupBy) > 0 {
			length := len(strings.Split(c.opt.GroupBy, "/"))
			matched, _ := doublestar.Match(filepath.Join(c.opt.GroupBy, "**"), change.Path)
			if !matched {
				log.Printf("[DEBUG] GetDirs: %s is not matched in %s\n", change.Path, c.opt.GroupBy)
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

	for _, arg := range c.args {
		dirs = dirs.Filter(func(dir Dir) bool {
			return strings.Index(dir.Path, arg) == 0
		})
	}

	for _, ignore := range c.opt.Ignores {
		dirs = dirs.Filter(func(dir Dir) bool {
			match, err := doublestar.Match(ignore, dir.Path)
			if err != nil {
				return false
			}
			return !match
		})
	}

	return dirs, nil
}
