package main

import (
	"fmt"
	"os"
	"os/exec"
)

const bgEnv = "WGET_BG"
const logFile = "wget-log"

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// Basic validation: something to do is required.
	if opts.InputFile == "" && len(opts.URLs) == 0 {
		fmt.Fprintln(os.Stderr, "error: no URL provided")
		os.Exit(1)
	}

	isChild := os.Getenv(bgEnv) == "1"

	// -B: relaunch ourselves detached, redirecting output to wget-log.
	if opts.Background && !isChild {
		if err := launchBackground(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Printf("Output will be written to %q.\n", logFile)
		return
	}

	// The background child writes to a file, so suppress the live bar.
	showProgress := !isChild

	if err := run(opts, showProgress); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run dispatches to the correct mode based on the parsed options.
func run(opts *Options, showProgress bool) error {
	switch {
	case opts.Mirror:
		if len(opts.URLs) == 0 {
			return fmt.Errorf("--mirror requires a URL")
		}
		return mirror(opts.URLs[0], opts)

	case opts.InputFile != "":
		return multiDownload(opts)

	default:
		for _, u := range opts.URLs {
			if _, err := downloadFile(u, opts, os.Stdout, showProgress); err != nil {
				return err
			}
		}
		return nil
	}
}

// launchBackground starts a detached copy of this process whose stdout and
// stderr are redirected to the log file.
func launchBackground() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	f, err := os.Create(logFile)
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), bgEnv+"=1")
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Stdin = nil

	return cmd.Start() // do not Wait: let it run in the background
}
