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

	DirChunk string

	OnlyDir bool
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

type ditto struct {
	args    []string
	opt     Option
	changes []git.Change
}

func New(path string, args []string, opt Option) (ditto, error) {
	changes, err := git.Open(git.Config{
		Path:          path,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
	})
	if err != nil {
		return ditto{}, err
	}
	return ditto{
		args:    args,
		opt:     opt,
		changes: changes,
	}, nil
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

func (d ditto) GetFiles() (Files, error) {
	var files Files

	for _, change := range d.changes {
		files = append(files, getFile(change))
	}

	if len(d.args) > 0 {
		var tmp Files
		for _, arg := range d.args {
			tmp = append(tmp, files.Filter(func(file File) bool {
				return strings.Index(file.Path, arg) == 0
			})...)
		}
		files = tmp
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

func (d ditto) GetDirs() ([]Dir, error) {
	data := make(map[string]Dir)
	chunk := d.opt.DirChunk
	length := len(strings.Split(chunk, "/"))

	for _, change := range d.changes {
		dir := change.Dir
		if len(chunk) > 0 {
			matched, _ := doublestar.Match(filepath.Join(chunk, "**"), change.Path)
			if !matched {
				log.Printf("[DEBUG] GetDirs: %s is not matched in %s\n", change.Path, chunk)
				continue
			}
			dir = strings.Join(strings.Split(change.Path, "/")[0:length], "/")
			log.Printf("[DEBUG] GetDirs: chunk dir %s\n", dir)
		}
		d, ok := data[dir]
		if ok {
			d.Files = append(d.Files, getFile(change))
		} else {
			d = Dir{
				Path:  dir,
				Files: Files{getFile(change)},
			}
		}
		data[dir] = d
	}

	var dirs Dirs
	for _, d := range data {
		dirs = append(dirs, d)
	}

	if len(d.args) > 0 {
		var tmp Dirs
		for _, arg := range d.args {
			tmp = append(tmp, dirs.Filter(func(dir Dir) bool {
				return strings.Index(dir.Path, arg) == 0
			})...)
		}
		dirs = tmp
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
