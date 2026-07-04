package main

import (
	"bytes"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// linkTargets is the fixed set of (tag, attribute) pairs the mirror follows,
// as required by the project: <a href>, <link href> and <img src>.
var linkTargets = map[string]string{
	"a":    "href",
	"link": "href",
	"img":  "src",
}

// cssURLRe matches url(...) references inside CSS, with optional quotes.
var cssURLRe = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)

// extractHTMLLinks parses HTML and returns the absolute URLs referenced by the
// followed tags, resolved against base.
func extractHTMLLinks(body []byte, base *url.URL) []string {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	var out []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if attrName, ok := linkTargets[n.Data]; ok {
				if v, ok := getAttr(n, attrName); ok {
					if abs := resolveRef(base, v); abs != "" {
						out = append(out, abs)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// extractCSSLinks returns absolute URLs referenced by url() in a CSS document.
func extractCSSLinks(body []byte, base *url.URL) []string {
	var out []string
	for _, m := range cssURLRe.FindAllSubmatch(body, -1) {
		if abs := resolveRef(base, string(m[1])); abs != "" {
			out = append(out, abs)
		}
	}
	return out
}

// resolveRef resolves a possibly-relative reference against base and strips the
// fragment. Empty, data:, javascript: and mailto: references are dropped.
func resolveRef(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") {
		return ""
	}
	low := strings.ToLower(ref)
	for _, skip := range []string{"data:", "javascript:", "mailto:", "tel:"} {
		if strings.HasPrefix(low, skip) {
			return ""
		}
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	resolved.Fragment = ""
	return resolved.String()
}

func getAttr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// rewriteHTMLFile re-reads an HTML file, converts every followed reference that
// was downloaded locally into a relative path, and writes the file back.
func rewriteHTMLFile(localFile string, pageURL *url.URL, resolve func(abs string) (string, bool)) error {
	data, err := os.ReadFile(localFile)
	if err != nil {
		return err
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return err
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if attrName, ok := linkTargets[n.Data]; ok {
				for i, a := range n.Attr {
					if a.Key != attrName {
						continue
					}
					abs := resolveRef(pageURL, a.Val)
					if abs == "" {
						continue
					}
					if rel, ok := resolve(abs); ok {
						n.Attr[i].Val = rel
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return err
	}
	return os.WriteFile(localFile, buf.Bytes(), 0o644)
}

// rewriteCSSFile converts url() references in a CSS file to local relative paths.
func rewriteCSSFile(localFile string, pageURL *url.URL, resolve func(abs string) (string, bool)) error {
	data, err := os.ReadFile(localFile)
	if err != nil {
		return err
	}
	out := cssURLRe.ReplaceAllFunc(data, func(match []byte) []byte {
		sub := cssURLRe.FindSubmatch(match)
		if sub == nil {
			return match
		}
		abs := resolveRef(pageURL, string(sub[1]))
		if abs == "" {
			return match
		}
		if rel, ok := resolve(abs); ok {
			return []byte("url(" + rel + ")")
		}
		return match
	})
	return os.WriteFile(localFile, out, 0o644)
}
