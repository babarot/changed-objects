package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/babarot/changed-objects/internal/detect"
	"github.com/hashicorp/logutils"
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
	GroupBy       []string `long:"group-by" description:"Specify a pattern to make into one group when showing changed objects"`
	DirExist      string   `long:"dir-exist" description:"Filter objects by state of dir existing" choice:"true" choice:"false" choice:"all" default:"all"`
	RootMarker    string   `long:"root-marker" description:"Specify a glob pattern of file that marks the root directory (e.g. *.tf)"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	out, err := logOutput()
	if err != nil {
		return err
	}

	if out == nil {
		out = io.Discard
	}
	log.SetOutput(out)

	defer log.Printf("[INFO] finish main function")

	log.Printf("[INFO] Version: %s (%s)", Version, Revision)
	log.Printf("[INFO] Args: %#v", args)

	var opt Option
	p := flags.NewParser(&opt, flags.HelpFlag|flags.PassDoubleDash)
	args, err = p.ParseArgs(args)
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
		DirExist:      opt.DirExist,
		RootMarker:    opt.RootMarker,
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

var ValidLevels = []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

func logOutput() (io.Writer, error) {
	out := io.Discard

	logLevel := LogLevel()
	if logLevel == "" {
		return out, nil
	}

	out = os.Stderr
	if logPath := os.Getenv("LOG_PATH"); logPath != "" {
		var err error
		out, err = os.OpenFile(logPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0666)
		if err != nil {
			return out, err
		}
	}

	out = &logutils.LevelFilter{
		Levels:   ValidLevels,
		MinLevel: logutils.LogLevel(logLevel),
		Writer:   out,
	}
	return out, nil
}

// LogLevel returns the current log level string based the environment vars
func LogLevel() string {
	envLevel := os.Getenv("LOG")
	if envLevel == "" {
		return ""
	}

	logLevel := "TRACE"
	if isValidLogLevel(envLevel) {
		// allow following for better ux: info, Info or INFO
		logLevel = strings.ToUpper(envLevel)
	} else {
		log.Printf("[WARN] Invalid log level: %q. Defaulting to level: TRACE. Valid levels are: %+v",
			envLevel, ValidLevels)
	}

	return logLevel
}

func isValidLogLevel(level string) bool {
	for _, l := range ValidLevels {
		if strings.ToUpper(level) == string(l) {
			return true
		}
	}

	return false
}
