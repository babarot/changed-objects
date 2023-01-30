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
	// Filters       []string `long:"filter" description:"Filter the kind of changed objects" default:"all" choice:"added" choice:"modified" choice:"deleted" choice:"all"`
	Dirname       bool   `long:"dirname" description:"Return changed objects with their directory name"`
	DirExist      bool   `long:"dir-exist" description:"Return changed objects if parent dir exists"`
	DirNotExist   bool   `long:"dir-not-exist" description:"Return changed objects if parent dir does not exist"`
	Output        string `long:"output" short:"o" description:"Format to output the result" default:"" choice:"json"`
	DefaultBranch string `long:"default-branch" description:"Specify default branch" default:"main"`
	MergeBase     string `long:"merge-base" description:"Specify merge-base revision"`

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

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := filepath.Abs(wd)
	if err != nil {
		return err
	}
	log.Printf("[INFO] git repo: %s", repo)

	switch {
	case opt.Version:
		fmt.Fprintf(os.Stdout, "%s (%s)\n", Version, Revision)
		return nil
	}

	stats, err := ditto.Get(repo, ditto.Option{
		Dirname:       opt.Dirname,
		DirExist:      opt.DirExist,
		DirNotExist:   opt.DirNotExist,
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
	}, args)

	switch opt.Output {
	case "json":
		r := struct {
			Repo  string      `json:"repo"`
			Stats ditto.Stats `json:"stats"`
		}{
			Repo:  repo,
			Stats: stats,
		}
		return json.NewEncoder(os.Stdout).Encode(&r)
	case "":
		for _, stat := range stats {
			fmt.Fprintln(os.Stdout, stat.Path)
		}
	default:
		return fmt.Errorf("%s: invalid output format", opt.Output)
	}

	return nil
}
