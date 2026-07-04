# wget (Go implementation)

A subset of GNU `wget` reimplemented in Go.

## Build / Run

```console
$ go run . <url>
```

Dependencies are vendored (`golang.org/x/net/html`), so it builds offline.

## Features

| Flag / usage                     | Description                                                     |
| -------------------------------- | -------------------------------------------------------------- |
| `go run . <url>`                 | Download a file, printing status, size, and a live progress bar |
| `-O=<name>`                      | Save the download under a different name                       |
| `-P=<path>`                      | Save the download into a specific directory                    |
| `--rate-limit=<n>` (`k`, `M`)    | Cap the download speed, e.g. `--rate-limit=400k`, `2M`         |
| `-B`                             | Download in the background; output goes to `wget-log`          |
| `-i=<file>`                      | Download every URL in a file, concurrently                     |
| `--mirror <url>`                 | Mirror an entire website into a folder named after the host    |
| `-R=<suffixes>` / `--reject`     | (mirror) Skip files with these suffixes, e.g. `-R=jpg,gif`     |
| `-X=<paths>` / `--exclude`       | (mirror) Skip these directory paths, e.g. `-X=/js,/assets`     |
| `--convert-links`                | (mirror) Rewrite links to point at the local copies            |

Flags accept both `-O=name` and `-O name` forms.

## Examples

```console
$ go run . https://pbs.twimg.com/media/EMtmPFLWkAA8CIS.jpg
$ go run . -O=meme.jpg https://example.com/file.jpg
$ go run . -P=~/Downloads/ -O=meme.jpg https://example.com/file.jpg
$ go run . --rate-limit=400k https://example.com/big.zip
$ go run . -B https://example.com/big.zip
$ go run . -i=download.txt
$ go run . --mirror https://example.com
$ go run . --mirror -R=jpg,gif https://example.com
$ go run . --mirror -X=/assets,/css https://example.com
$ go run . --mirror --convert-links https://example.com
```

## Source layout

- `main.go` — entry point, flag dispatch, `-B` background relaunch
- `options.go` — command-line parsing (`-flag=value` and `-flag value`)
- `download.go` — single-file download + wget-style report
- `progress.go` — live progress bar
- `ratelimit.go` — throughput limiter for `--rate-limit`
- `multi.go` — concurrent `-i` downloads
- `mirror.go` — website mirroring, `-R`, `-X`
- `links.go` — HTML/CSS link extraction and `--convert-links` rewriting
- `format.go` — time and byte-size formatting helpers
