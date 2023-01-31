package ditto

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/b4b4r07/changed-objects/git"
	"github.com/bmatcuk/doublestar"
)

type File struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Kind      git.Kind  `json:"kind"`
	ParentDir ParentDir `json:"parent_dir"`
}

type ParentDir struct {
	Path  string `json:"path"`
	Exist bool   `json:"exist"`
}

type Dir struct {
	Path  string `json:"path"`
	Files []File `json:"files"`
}

// Stat represents the stats for a file in a commit.
type Stat struct {
	Kind     git.Kind `json:"kind"`
	Path     string   `json:"path"`
	DirExist bool     `json:"dir-exist"`
}

type Stats []Stat

type Option struct {
	DirExist      bool
	DirNotExist   bool
	DefaultBranch string
	MergeBase     string

	DirChunk string

	OnlyDir bool
}

func GetFiles(path string, args []string, opt Option) ([]File, error) {
	var files []File

	changes, err := git.Open(git.Config{
		Path:          path,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
	})
	if err != nil {
		return []File{}, err
	}

	for _, change := range changes {
		files = append(files, getFile(change))
	}

	// if len(args) > 0 {
	// 	var ss Stats
	// 	log.Printf("[TRACE] Filtering with args")
	// 	for _, arg := range args {
	// 		ss = append(ss, stats.Filter(func(stat Stat) bool {
	// 			log.Printf("[TRACE] Filtering with stat %q, file %q", stat.Path, arg)
	// 			return strings.Index(stat.Path, arg) == 0
	// 		})...)
	// 	}
	// 	stats = ss
	// }

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

func GetDirs(path string, args []string, opt Option) ([]Dir, error) {
	var dirs []Dir

	changes, err := git.Open(git.Config{
		Path:          path,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
	})
	if err != nil {
		return []Dir{}, err
	}

	data := make(map[string]Dir)
	chunk := opt.DirChunk
	length := len(strings.Split(chunk, "/"))

	for _, change := range changes {
		dir := change.Dir
		if len(chunk) > 0 {
			matched, _ := doublestar.Match(filepath.Join(chunk, "**"), change.Path)
			if !matched {
				continue
			}
			dir = strings.Join(strings.Split(change.Path, "/")[0:length], "/")
		}
		d, ok := data[dir]
		if ok {
			d.Files = append(d.Files, getFile(change))
		} else {
			d = Dir{
				Path:  dir,
				Files: []File{getFile(change)},
			}
		}
		data[dir] = d
	}

	for _, d := range data {
		dirs = append(dirs, d)
	}

	// if len(args) > 0 {
	// 	var ss Stats
	// 	log.Printf("[TRACE] Filtering with args")
	// 	for _, arg := range args {
	// 		ss = append(ss, stats.Filter(func(stat Stat) bool {
	// 			log.Printf("[TRACE] Filtering with stat %q, file %q", stat.Path, arg)
	// 			return strings.Index(stat.Path, arg) == 0
	// 		})...)
	// 	}
	// 	stats = ss
	// }

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

func Get(fp string, args []string, opt Option) (Stats, error) {
	var stats Stats

	files, err := git.Open(git.Config{
		Path:          fp,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
	})
	if err != nil {
		return stats, err
	}

	for _, file := range files {
		stats = append(stats, Stat{
			Kind: file.Kind,
			Path: file.Path,
			DirExist: func() bool {
				_, err := os.Stat(filepath.Dir(file.Path))
				return err == nil
			}(),
		})
	}

	if len(args) > 0 {
		var ss Stats
		log.Printf("[TRACE] Filtering with args")
		for _, arg := range args {
			ss = append(ss, stats.Filter(func(stat Stat) bool {
				log.Printf("[TRACE] Filtering with stat %q, file %q", stat.Path, arg)
				return strings.Index(stat.Path, arg) == 0
			})...)
		}
		stats = ss
	}

	if opt.DirExist {
		stats = stats.Filter(func(stat Stat) bool {
			return stat.DirExist
		})
	}

	if opt.DirNotExist {
		stats = stats.Filter(func(stat Stat) bool {
			return !stat.DirExist
		})
	}

	// OnlyDir
	if opt.OnlyDir {
		stats = stats.Dirs()
	}

	return stats, nil
}

func (ss *Stats) Filter(f func(Stat) bool) Stats {
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if f(stat) {
			stats = append(stats, stat)
		}
	}
	return stats
}

func (ss *Stats) Map(f func(Stat) Stat) Stats {
	stats := make([]Stat, len(*ss))
	for i, stat := range *ss {
		stats[i] = f(stat)
	}
	return stats
}

func (ss *Stats) Dirs() Stats {
	m := make(map[string]bool)
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		dir := filepath.Dir(stat.Path)
		exists := func() bool {
			_, err := os.Stat(dir)
			return err == nil
		}()
		kind := git.Deletion
		if exists {
			kind = git.Modification
		}
		if !m[dir] {
			m[dir] = true
			stats = append(stats, Stat{
				Kind:     kind,
				Path:     dir,
				DirExist: exists,
			})
		}
	}
	return stats
}
