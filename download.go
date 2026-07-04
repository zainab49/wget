package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// httpClient is shared so connections can be reused across downloads.
var httpClient = &http.Client{}

// fileNameFromURL derives a sensible file name from a URL, falling back to
// "index.html" when the path has no basename.
func fileNameFromURL(rawURL string) string {
	u := rawURL
	if i := strings.IndexAny(u, "?#"); i != -1 {
		u = u[:i]
	}
	name := path.Base(u)
	if name == "" || name == "." || name == "/" {
		return "index.html"
	}
	return name
}

// resolvePaths returns the on-disk directory to write to, the actual file path,
// and the human-facing display path that mirrors what wget prints.
func resolvePaths(rawURL string, opts *Options) (actualPath, displayPath string) {
	name := opts.OutputName
	if name == "" {
		name = fileNameFromURL(rawURL)
	}

	if opts.Path == "" {
		return filepath.Join(".", name), "./" + name
	}

	rawDir := opts.Path
	displayPath = strings.TrimRight(rawDir, "/\\") + "/" + name
	actualDir := expandHome(rawDir)
	return filepath.Join(actualDir, name), displayPath
}

// downloadFile performs a single download and prints the standard wget-style
// report to out. When showProgress is true a live progress bar is rendered.
// It returns the content size actually written.
func downloadFile(rawURL string, opts *Options, out io.Writer, showProgress bool) (int64, error) {
	fmt.Fprintf(out, "start at %s\n", now())

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Wget/1.0 (go)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	fmt.Fprintf(out, "sending request, awaiting response... status %s\n", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	size := resp.ContentLength
	if size >= 0 {
		fmt.Fprintf(out, "content size: %d [%s]\n", size, humanDecimal(size))
	} else {
		fmt.Fprintf(out, "content size: unknown\n")
	}

	actualPath, displayPath := resolvePaths(rawURL, opts)
	if dir := filepath.Dir(actualPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, err
		}
	}
	fmt.Fprintf(out, "saving file to: %s\n", displayPath)

	f, err := os.Create(actualPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var reader io.Reader = resp.Body
	reader = newRateLimitedReader(reader, opts.RateLimit)

	var written int64
	if showProgress {
		bar := newProgressBar(out, size)
		written, err = io.Copy(io.MultiWriter(f, bar), reader)
		bar.Finish()
	} else {
		written, err = io.Copy(f, reader)
	}
	if err != nil {
		return written, err
	}

	// A blank line separates the finished progress bar from the summary; when
	// there is no bar (background mode) we skip it to match wget's log output.
	if showProgress {
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "Downloaded [%s]\n", rawURL)
	fmt.Fprintf(out, "finished at %s\n", now())
	return written, nil
}
