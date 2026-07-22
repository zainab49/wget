package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// multiDownload reads URLs from opts.InputFile and downloads them concurrently.
func multiDownload(opts *Options) error {
	urls, err := readURLList(opts.InputFile)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return fmt.Errorf("no URLs found in %s", opts.InputFile)
	}

	// Open one GET per URL concurrently. The response headers already carry
	// Content-Length, so there is no need for a separate size-probe request.
	resps := make([]*http.Response, len(urls))
	errs := make([]error, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			resps[i], errs[i] = httpClient.Get(u)
		}(i, u)
	}
	wg.Wait()

	sizeStrs := make([]string, len(urls))
	for i := range urls {
		if errs[i] == nil && resps[i] != nil && resps[i].ContentLength >= 0 {
			sizeStrs[i] = fmt.Sprintf("%d", resps[i].ContentLength)
		} else {
			sizeStrs[i] = "unknown"
		}
	}
	fmt.Printf("content size: [%s]\n", strings.Join(sizeStrs, ", "))

	// Now stream every response body to disk concurrently.
	var (
		mu       sync.Mutex
		firstErr error
	)
	wg = sync.WaitGroup{}
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			name, err := saveResponse(u, resps[i], errs[i], opts)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fmt.Printf("error downloading %s: %v\n", u, err)
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			fmt.Printf("finished %s\n", name)
		}(i, u)
	}
	wg.Wait()

	fmt.Printf("\nDownload finished: %v\n", urls)
	return firstErr
}

// readURLList loads non-empty, non-comment lines from a file, tolerating a
// leading UTF-8 byte-order mark.
func readURLList(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const bom = "\ufeff"
	var urls []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), bom))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, sc.Err()
}

// saveResponse writes an already-open response body to disk and returns the
// saved file's base name. It owns closing the body. Used by the -i concurrent
// path, where the same GET that reported the size also delivers the content.
func saveResponse(rawURL string, resp *http.Response, getErr error, opts *Options) (string, error) {
	if getErr != nil {
		return "", getErr
	}
	if resp == nil {
		return "", fmt.Errorf("no response")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	name := fileNameFromURL(rawURL)
	dir := "."
	if opts.Path != "" {
		dir = expandHome(opts.Path)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, name)

	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := newRateLimitedReader(resp.Body, opts.RateLimit)
	if _, err := io.Copy(f, reader); err != nil {
		return "", err
	}
	return name, nil
}
