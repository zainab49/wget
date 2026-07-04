# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.23-alpine AS build

WORKDIR /src

# Copy module files first so dependency layers cache independently of source.
COPY go.mod go.sum ./

# Copy the vendored deps and source, then build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /out/wget .

# ---- Runtime stage ----
FROM alpine:3.20

# CA certificates are needed for HTTPS downloads.
RUN apk add --no-cache ca-certificates

COPY --from=build /out/wget /usr/local/bin/wget

# Downloads are written to the working directory; mount a host volume here.
WORKDIR /downloads

ENTRYPOINT ["wget"]
