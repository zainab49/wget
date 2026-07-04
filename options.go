package main

import (
	"fmt"
	"os"
	"strings"
)

// Options holds every parsed command-line setting.
type Options struct {
	Background   bool     // -B
	OutputName   string   // -O=name
	Path         string   // -P=path
	RateLimit    int64    // --rate-limit=400k (bytes/sec, 0 = unlimited)
	InputFile    string   // -i=file
	Mirror       bool     // --mirror
	Reject       []string // -R / --reject   (suffixes)
	Exclude      []string // -X / --exclude  (paths)
	ConvertLinks bool     // --convert-links
	URLs         []string // positional URLs
}

// parseArgs turns the raw argument slice into Options. It accepts both the
// "-flag=value" form shown in the project examples and the "-flag value" form.
func parseArgs(args []string) (*Options, error) {
	opts := &Options{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Positional argument (a URL).
		if !strings.HasPrefix(arg, "-") {
			opts.URLs = append(opts.URLs, arg)
			continue
		}

		// Split "name=value" once.
		name, value, hasValue := arg, "", false
		if idx := strings.Index(arg, "="); idx != -1 {
			name = arg[:idx]
			value = arg[idx+1:]
			hasValue = true
		}

		// needValue fetches the value either from "=..." or the next arg.
		needValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if i+1 < len(args) {
				i++
				return args[i], nil
			}
			return "", fmt.Errorf("flag %s requires a value", name)
		}

		switch name {
		case "-B":
			opts.Background = true
		case "--mirror":
			opts.Mirror = true
		case "--convert-links":
			opts.ConvertLinks = true
		case "-O":
			v, err := needValue()
			if err != nil {
				return nil, err
			}
			opts.OutputName = v
		case "-P":
			v, err := needValue()
			if err != nil {
				return nil, err
			}
			opts.Path = v
		case "--rate-limit":
			v, err := needValue()
			if err != nil {
				return nil, err
			}
			rate, err := parseRate(v)
			if err != nil {
				return nil, err
			}
			opts.RateLimit = rate
		case "-i":
			v, err := needValue()
			if err != nil {
				return nil, err
			}
			opts.InputFile = v
		case "-R", "--reject":
			v, err := needValue()
			if err != nil {
				return nil, err
			}
			opts.Reject = append(opts.Reject, splitList(v)...)
		case "-X", "--exclude":
			v, err := needValue()
			if err != nil {
				return nil, err
			}
			opts.Exclude = append(opts.Exclude, splitList(v)...)
		default:
			return nil, fmt.Errorf("unknown flag: %s", name)
		}
	}

	return opts, nil
}

// splitList splits a comma-separated flag value and trims blanks.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// parseRate converts values like "400k", "2M" or "1024" into bytes per second.
func parseRate(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty rate limit")
	}

	mult := int64(1)
	switch s[len(s)-1] {
	case 'k', 'K':
		mult = 1024
		s = s[:len(s)-1]
	case 'm', 'M':
		mult = 1024 * 1024
		s = s[:len(s)-1]
	case 'g', 'G':
		mult = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	var n float64
	if _, err := fmt.Sscanf(s, "%f", &n); err != nil {
		return 0, fmt.Errorf("invalid rate limit: %q", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("rate limit must be positive")
	}
	return int64(n * float64(mult)), nil
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:]
		}
	}
	return p
}
