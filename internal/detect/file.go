package detect

import (
	"os"
	"path/filepath"

	"github.com/b4b4r07/changed-objects/internal/git"
)

type File struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Type      git.Type  `json:"type"`
	ParentDir ParentDir `json:"parent_dir"`
}

type Files []File

type ParentDir struct {
	Path  string `json:"path"`
	Exist bool   `json:"exist"`
}

type Dir struct {
	Path  string `json:"path"`
	Exist bool   `json:"exist"`
	Files Files  `json:"files"`
}

type Dirs []Dir

func (fs *Files) filter(f func(File) bool) Files {
	files := make(Files, 0)
	for _, file := range *fs {
		if f(file) {
			files = append(files, file)
		}
	}
	return files
}

func (ds *Dirs) filter(f func(Dir) bool) Dirs {
	dirs := make(Dirs, 0)
	for _, dir := range *ds {
		if f(dir) {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

type Diff struct {
	Files Files `json:"files"`
	Dirs  Dirs  `json:"dirs"`
}

func getFile(change git.Change) File {
	return File{
		Name: filepath.Base(change.Path),
		Path: change.Path,
		Type: change.Type,
		ParentDir: ParentDir{
			Path: change.Dir,
			Exist: func() bool {
				_, err := os.Stat(change.Dir)
				return err == nil
			}(),
		},
	}
}
