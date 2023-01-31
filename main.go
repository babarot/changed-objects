package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/b4b4r07/changed-objects/ditto"
	clilog "github.com/b4b4r07/go-cli-log"
	"github.com/jessevdk/go-flags"
)

var (
	Version  = "unset"
	Revision = "unset"
)

type Option struct {
	Filters       []string `long:"filter" description:"Filter the kind of changed objects" default:"all" choice:"added" choice:"modified" choice:"deleted" choice:"all"`
	OnlyDir       bool     `long:"only-dir" description:"Return changed objects with their directory name"`
	DirExist      bool     `long:"dir-exist" description:"Return changed objects if parent dir exists"`
	DirNotExist   bool     `long:"dir-not-exist" description:"Return changed objects if parent dir does not exist"`
	Output        string   `long:"output" short:"o" description:"Format to output the result" default:"" choice:"json"`
	DefaultBranch string   `long:"default-branch" description:"Specify default branch" default:"main"`
	MergeBase     string   `long:"merge-base" description:"Specify merge-base revision"`

	Ignores []string `long:"ignore" description:"Ignore string pattern"`

	Version bool `short:"v" long:"version" description:"Show version"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	clilog.Env = "LOG"
	clilog.SetOutput()
	defer log.Printf("[INFO] finish main function")

	log.Printf("[INFO] Version: %s (%s)", Version, Revision)
	log.Printf("[INFO] Args: %#v", args)

	var opt Option
	args, err := flags.ParseArgs(&opt, args)
	if err != nil {
		return err
	}

	switch {
	case opt.Version:
		fmt.Fprintf(os.Stdout, "%s (%s)\n", Version, Revision)
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := filepath.Abs(wd)
	if err != nil {
		return err
	}
	log.Printf("[INFO] git repo: %s", repo)

	// stats, err := ditto.Get(repo, args, ditto.Option{
	// 	DirExist:      opt.DirExist,
	// 	DirNotExist:   opt.DirNotExist,
	// 	DefaultBranch: opt.DefaultBranch,
	// 	MergeBase:     opt.MergeBase,
	// 	OnlyDir:       opt.OnlyDir,
	// })
	//
	// log.Printf("[INFO] Option filters: %#v", opt.Filters)
	// var ss ditto.Stats
	// for _, filter := range opt.Filters {
	// 	switch filter {
	// 	case "all":
	// 		ss = stats
	// 		break
	// 	case "added":
	// 		ss = append(ss, stats.Filter(func(stat ditto.Stat) bool {
	// 			return stat.Kind == git.Addition
	// 		})...)
	// 	case "deleted":
	// 		ss = append(ss, stats.Filter(func(stat ditto.Stat) bool {
	// 			return stat.Kind == git.Deletion
	// 		})...)
	// 	case "modified":
	// 		ss = append(ss, stats.Filter(func(stat ditto.Stat) bool {
	// 			return stat.Kind == git.Modification
	// 		})...)
	// 	case "":
	// 		return fmt.Errorf("requires a filter at least one")
	// 	}
	// }
	// stats = ss
	//
	// for _, ignore := range opt.Ignores {
	// 	stats = stats.Filter(func(stat ditto.Stat) bool {
	// 		match, err := doublestar.Match(ignore, stat.Path)
	// 		if err != nil {
	// 			fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
	// 			return false
	// 		}
	// 		return !match
	// 	})
	// }
	//
	// switch opt.Output {
	// case "json":
	// 	r := struct {
	// 		Repo  string      `json:"repo"`
	// 		Stats ditto.Stats `json:"stats"`
	// 	}{
	// 		Repo:  repo,
	// 		Stats: stats,
	// 	}
	// 	return json.NewEncoder(os.Stdout).Encode(&r)
	// case "":
	// 	// Remove redundants
	// 	paths := make(map[string]bool)
	// 	for _, stat := range stats {
	// 		if !paths[stat.Path] {
	// 			paths[stat.Path] = true
	// 		}
	// 	}
	// 	for path := range paths {
	// 		fmt.Fprintln(os.Stdout, path)
	// 	}
	// default:
	// 	return fmt.Errorf("%s: invalid output format", opt.Output)
	// }

	dopt := ditto.Option{
		DirExist:      opt.DirExist,
		DirNotExist:   opt.DirNotExist,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
		OnlyDir:       opt.OnlyDir,
	}

	files, err := ditto.GetFile(repo, args, dopt)
	if err != nil {
		return err
	}

	dirs, err := ditto.GetDirs(repo, args, dopt)
	if err != nil {
		return err
	}

	r := struct {
		Repo  string       `json:"repo"`
		Files []ditto.File `json:"files"`
		Dirs  []ditto.Dir  `json:"dirs"`
	}{
		Repo:  repo,
		Files: files,
		Dirs:  dirs,
	}
	return json.NewEncoder(os.Stdout).Encode(&r)
}
