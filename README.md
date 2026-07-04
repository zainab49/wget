# wget

A subset of GNU [`wget`](https://www.gnu.org/software/wget/) reimplemented in Go.
It downloads files over HTTP/HTTPS with a live progress bar and supports rate
limiting, background downloads, batch downloads from a file, and full website
mirroring with link rewriting.

## Features

- Download a single file with a wget-style status report and live progress bar
- Save under a custom name (`-O`) or into a specific directory (`-P`)
- Cap download speed with `--rate-limit` (`k` / `M` suffixes)
- Download in the background (`-B`), logging to `wget-log`
- Download many URLs concurrently from a list file (`-i`)
- Mirror an entire website (`--mirror`), with suffix rejection (`-R`),
  path exclusion (`-X`), and link conversion (`--convert-links`)

Dependencies are vendored (`golang.org/x/net/html`), so the project builds
offline with no network access.

## Requirements

- [Go](https://go.dev/dl/) 1.23 or newer (to build from source), **or**
- [Docker](https://www.docker.com/) (to run in a container)

## Build & Run

### From source

```console
$ go build -o wget .
$ ./wget https://example.com/file.jpg
```

Or run without building a binary:

```console
$ go run . https://example.com/file.jpg
```

### With Docker

Build the image:

```console
$ docker build -t wget-go .
```

Run it, mounting the current directory so downloads land on the host:

```console
$ docker run --rm -v "$PWD:/downloads" wget-go https://example.com/file.jpg
```

The container's working directory is `/downloads`, so files are written to
whatever host directory you mount there.

## Usage

```console
$ wget <url>
```

| Flag / usage                     | Description                                                      |
| -------------------------------- | --------------------------------------------------------------- |
| `<url>`                          | Download a file, printing status, size, and a live progress bar |
| `-O=<name>`                      | Save the download under a different name                        |
| `-P=<path>`                      | Save the download into a specific directory                     |
| `--rate-limit=<n>` (`k`, `M`)    | Cap the download speed, e.g. `--rate-limit=400k`, `2M`          |
| `-B`                             | Download in the background; output goes to `wget-log`           |
| `-i=<file>`                      | Download every URL in a file, concurrently                      |
| `--mirror <url>`                 | Mirror an entire website into a folder named after the host     |
| `-R=<suffixes>` / `--reject`     | (mirror) Skip files with these suffixes, e.g. `-R=jpg,gif`      |
| `-X=<paths>` / `--exclude`       | (mirror) Skip these directory paths, e.g. `-X=/js,/assets`      |
| `--convert-links`                | (mirror) Rewrite links to point at the local copies             |

Flags accept both `-O=name` and `-O name` forms.

## Examples

```console
$ wget https://example.com/file.jpg
$ wget -O=meme.jpg https://example.com/file.jpg
$ wget -P=~/Downloads/ -O=meme.jpg https://example.com/file.jpg
$ wget --rate-limit=400k https://example.com/big.zip
$ wget -B https://example.com/big.zip
$ wget -i=download.txt
$ wget --mirror https://example.com
$ wget --mirror -R=jpg,gif https://example.com
$ wget --mirror -X=/assets,/css https://example.com
$ wget --mirror --convert-links https://example.com
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

See [USAGE.md](USAGE.md) for additional detail.
