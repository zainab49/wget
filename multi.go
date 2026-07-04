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

	// First, fetch content sizes concurrently so we can print them together.
	sizes := make([]int64, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			sizes[i] = contentLength(u)
		}(i, u)
	}
	wg.Wait()

	sizeStrs := make([]string, len(sizes))
	for i, s := range sizes {
		if s >= 0 {
			sizeStrs[i] = fmt.Sprintf("%d", s)
		} else {
			sizeStrs[i] = "unknown"
		}
	}
	fmt.Printf("content size: [%s]\n", strings.Join(sizeStrs, ", "))

	// Now download every file concurrently.
	var (
		mu       sync.Mutex
		firstErr error
	)
	wg = sync.WaitGroup{}
	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			name, err := saveURL(u, opts)
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
		}(u)
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

// contentLength performs a lightweight request to discover a file's size.
// It returns -1 when the size is unknown.
func contentLength(rawURL string) int64 {
	resp, err := httpClient.Head(rawURL)
	if err != nil || resp.StatusCode != http.StatusOK || resp.ContentLength < 0 {
		if resp != nil {
			resp.Body.Close()
		}
		// Fall back to a GET (some servers reject HEAD).
		resp, err = httpClient.Get(rawURL)
		if err != nil {
			return -1
		}
		defer resp.Body.Close()
		return resp.ContentLength
	}
	resp.Body.Close()
	return resp.ContentLength
}

// saveURL downloads a single URL quietly (no report) and returns the saved
// file's base name. Used by the -i concurrent path.
func saveURL(rawURL string, opts *Options) (string, error) {
	resp, err := httpClient.Get(rawURL)
	if err != nil {
		return "", err
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
