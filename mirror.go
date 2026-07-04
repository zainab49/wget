package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// page records a downloaded document we may need to rewrite later.
type page struct {
	url   *url.URL
	local string // path on disk relative to the working directory
	isCSS bool
}

// mirrorer crawls a website starting from a base URL and stores it under a
// folder named after the host.
type mirrorer struct {
	base    *url.URL
	opts    *Options
	visited map[string]bool
	local   map[string]string // normalized abs URL -> local relative path
	pages   []page
}

// mirror is the entry point for the --mirror flag.
func mirror(rawURL string, opts *Options) error {
	base, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if base.Scheme == "" {
		base.Scheme = "http"
	}
	if base.Host == "" {
		return fmt.Errorf("invalid mirror URL: %s", rawURL)
	}

	m := &mirrorer{
		base:    base,
		opts:    opts,
		visited: map[string]bool{},
		local:   map[string]string{},
	}

	fmt.Printf("start at %s\n", now())
	fmt.Printf("mirroring %s into ./%s/\n", base.String(), m.hostDir())

	queue := []*url.URL{base}
	m.visited[normalize(base)] = true

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		discovered, err := m.process(cur)
		if err != nil {
			fmt.Printf("skip %s: %v\n", cur, err)
			continue
		}

		for _, d := range discovered {
			du, err := url.Parse(d)
			if err != nil {
				continue
			}
			key := normalize(du)
			if m.visited[key] {
				continue
			}
			if !m.shouldFollow(du) {
				continue
			}
			m.visited[key] = true
			queue = append(queue, du)
		}
	}

	if opts.ConvertLinks {
		m.convertLinks()
	}

	fmt.Printf("finished at %s\n", now())
	return nil
}

// process downloads one URL, saves it, and returns the links it references.
func (m *mirrorer) process(u *url.URL) ([]string, error) {
	resp, err := httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}

	ctype := resp.Header.Get("Content-Type")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	localRel := m.localPath(u, ctype)
	if err := writeFile(localRel, body); err != nil {
		return nil, err
	}
	fmt.Printf("saved %s\n", localRel)
	m.local[normalize(u)] = localRel

	isHTML := strings.Contains(ctype, "html") ||
		strings.HasSuffix(localRel, ".html") || strings.HasSuffix(localRel, ".htm")
	isCSS := strings.Contains(ctype, "css") || strings.HasSuffix(localRel, ".css")

	if isHTML {
		m.pages = append(m.pages, page{url: u, local: localRel, isCSS: false})
		return extractHTMLLinks(body, u), nil
	}
	if isCSS {
		m.pages = append(m.pages, page{url: u, local: localRel, isCSS: true})
		return extractCSSLinks(body, u), nil
	}
	return nil, nil
}

// shouldFollow applies same-host, --exclude and --reject rules.
func (m *mirrorer) shouldFollow(u *url.URL) bool {
	if u.Host != m.base.Host {
		return false
	}
	if m.isExcluded(u.Path) {
		return false
	}
	if m.isRejected(u.Path) {
		return false
	}
	return true
}

// isExcluded reports whether the path falls under an -X excluded directory.
func (m *mirrorer) isExcluded(p string) bool {
	if p == "" {
		p = "/"
	}
	for _, ex := range m.opts.Exclude {
		ex = "/" + strings.Trim(ex, "/")
		if p == ex || strings.HasPrefix(p, ex+"/") {
			return true
		}
	}
	return false
}

// isRejected reports whether the path ends with an -R rejected suffix.
func (m *mirrorer) isRejected(p string) bool {
	base := path.Base(p)
	for _, suf := range m.opts.Reject {
		suf = strings.TrimPrefix(suf, ".")
		if strings.HasSuffix(strings.ToLower(base), "."+strings.ToLower(suf)) {
			return true
		}
	}
	return false
}

// localPath maps a URL to a path on disk under the host directory.
func (m *mirrorer) localPath(u *url.URL, ctype string) string {
	p := u.Path
	if p == "" {
		p = "/"
	}
	// Directory-style URLs become index.html.
	if strings.HasSuffix(p, "/") {
		p += "index.html"
	} else if path.Ext(p) == "" && strings.Contains(ctype, "html") {
		// Extension-less HTML pages get a .html suffix so browsers open them.
		p += ".html"
	}
	return filepath.Join(m.hostDir(), filepath.FromSlash(p))
}

// hostDir is the top-level folder name for the mirror. It equals the URL host
// (e.g. "www.example.com"); on Windows the port colon is swapped for "+" since
// ":" is not a legal path character there.
func (m *mirrorer) hostDir() string {
	host := m.base.Host
	if runtime.GOOS == "windows" {
		host = strings.ReplaceAll(host, ":", "+")
	}
	return host
}

// convertLinks rewrites references in every downloaded HTML/CSS file to point
// at the local copies.
func (m *mirrorer) convertLinks() {
	for _, pg := range m.pages {
		fromDir := filepath.Dir(pg.local)
		resolver := func(abs string) (string, bool) {
			target, ok := m.local[abs]
			if !ok {
				return "", false
			}
			rel, err := filepath.Rel(fromDir, target)
			if err != nil {
				return "", false
			}
			return filepath.ToSlash(rel), true
		}

		var err error
		if pg.isCSS {
			err = rewriteCSSFile(pg.local, pg.url, resolver)
		} else {
			err = rewriteHTMLFile(pg.local, pg.url, resolver)
		}
		if err != nil {
			fmt.Printf("convert-links failed for %s: %v\n", pg.local, err)
		}
	}
	fmt.Println("converted links for offline viewing")
}

// normalize returns a stable key for a URL (scheme+host+path+query, no fragment).
func normalize(u *url.URL) string {
	c := *u
	c.Fragment = ""
	return c.String()
}

// writeFile creates parent directories then writes the file.
func writeFile(rel string, data []byte) error {
	if dir := filepath.Dir(rel); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(rel, data, 0o644)
}
