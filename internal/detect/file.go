package detect

import (
	"os"
	"path/filepath"

	"github.com/babarot/changed-objects/internal/git"
)

type File struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Type      git.Type  `json:"type"`
	ParentDir ParentDir `json:"parent_dir"`
}

type ParentDir struct {
	Path  string `json:"path"`
	Exist bool   `json:"exist"`
}

type Dir struct {
	Path  string `json:"path"`
	Exist bool   `json:"exist"`
	Files []File `json:"files"`
}

type Diff struct {
	Files []File `json:"files"`
	Dirs  []Dir  `json:"dirs"`
}

func getFile(change git.Change) File {
	return File{
		Name: filepath.Base(change.Path),
		Path: change.Path,
		Type: change.Type,
		ParentDir: ParentDir{
			Path: filepath.Dir(change.Path),
			Exist: func() bool {
				_, err := os.Stat(filepath.Dir(change.Path))
				return err == nil
			}(),
		},
	}
}
