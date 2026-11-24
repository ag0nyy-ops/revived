# gohttpbench

A rebuild of [parkghost/gohttpbench](https://github.com/parkghost/gohttpbench), an ApacheBench-style HTTP load tester written in Go.

- **Original:** parkghost/gohttpbench by Brandon Chen, MIT licensed
- **This rebuild:** written from scratch in Go, standard library only, MIT licensed. No original source was copied; the original is credited for the idea and the `ab`-like option set.

## Why the original died

The repository was archived in 2023. It was built on the pre-modules `go get` workflow, kept latencies in whole milliseconds, printed nothing if you interrupted a long run, could only run a fixed request count, and never shipped binaries (its last open issue is "Build binary outputs").

## What's improved

- Standard library only, Go modules, current Go
- Latency kept at full resolution; min/mean/sd/median/max and the percentile table print fractional milliseconds (#35)
- Ctrl-C stops the workers and prints a report for what finished, marked as interrupted (#36)
- `-d 10s` duration mode and `--json` output for scripting (#37)
- Failures split into `connect`, `timeout`, `read`, `length` and `status`, plus per-status-code counts (#38)
- Connection errors stop the run unless `-r` is given; exit code is 1 when any request failed, 2 on usage errors
- Redirects are not followed, so you measure the URL you gave
- Tests run against `httptest` servers, including cancellation and failure paths

## Install

    go install github.com/ag0nyy-ops/revived/gohttpbench@latest

## Usage

    gohttpbench [options] http[s]://host[:port]/path

| flag | meaning |
|---|---|
| `-n N` | number of requests (default 1) |
| `-c N` | concurrent workers (default 1) |
| `-d DUR` | run for a duration such as `10s` instead of `-n` |
| `-t DUR` | per-request timeout (default 30s) |
| `-k` | HTTP keep-alive |
| `-z` | request gzip |
| `-i` | HEAD instead of GET |
| `-p FILE` / `-u FILE` | POST / PUT the file's contents |
| `-T TYPE` | content type for bodies (default text/plain) |
| `-H 'K: v'` | extra header, repeatable (`Host:` overrides the host header) |
| `-C 'k=v'` | cookie, repeatable |
| `-A user:pass` | basic auth |
| `-r` | keep going after connection errors |
| `-q` | no progress output |
| `--json` | JSON report on stdout |

Example:

    $ gohttpbench -n 10000 -c 50 -k http://localhost:8080/
    $ gohttpbench -d 10s -c 20 --json http://localhost:8080/ | jq .requests_per_second

## Tests

    go test -race ./...
