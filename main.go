package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/b4b4r07/changed-objects/internal/detect"
	clilog "github.com/b4b4r07/go-cli-log"
	"github.com/jessevdk/go-flags"
)

var (
	Version  = "unset"
	Revision = "unset"
)

type Option struct {
	Version bool `short:"v" long:"version" description:"Show version"`

	DefaultBranch string   `long:"default-branch" short:"b" description:"Specify default branch name" default:"main"`
	MergeBase     string   `long:"merge-base" short:"m" description:"Specify a Git reference as good common ancestors as possible for a merge"`
	Types         []string `long:"type" description:"Specify the type of changed objects" choice:"added" choice:"modified" choice:"deleted"`
	Ignores       []string `long:"ignore" description:"Specify a pattern to skip when showing changed objects"`
	GroupBy       string   `long:"group-by" description:"Specify a pattern to make into one group when showing changed objects"`
	ParentDir     string   `long:"parent-dir" description:"Filter objects by state of parent dir" choice:"exist" choice:"deleted" choice:"all" default:"all"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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
	p := flags.NewParser(&opt, flags.HelpFlag|flags.PassDoubleDash)
	args, err := p.ParseArgs(args)
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

	d, err := detect.New(repo, args, detect.Option{
		DefaultBranch: opt.DefaultBranch,
		MergeBase:     opt.MergeBase,
		Ignores:       opt.Ignores,
		GroupBy:       opt.GroupBy,
		Types:         opt.Types,
		ParentDir:     opt.ParentDir,
	})
	if err != nil {
		return err
	}

	diff, err := d.Run()
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(&diff)
}
