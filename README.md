# revived

Rebuilt and improved versions of abandoned open-source tools.

Each folder is a from-scratch rewrite of a tool that was archived, deprecated or left unmaintained. Only permissively licensed originals (MIT, Apache-2.0, BSD, ISC, Unlicense) are picked, every rebuild credits the original, and no original code is copied. The goal for each one: modern runtime, known issues fixed, simpler to use, and tests that actually run.

## Tools

| tool | original | what's new |
|---|---|---|
| [chglog](chglog/) | [git-chglog/git-chglog](https://github.com/git-chglog/git-chglog) (Go, MIT, deprecated) | zero-config, auto next version (`--bump`), graph-correct release grouping, revert cancelling, monorepo tags and paths, JSON output, zero deps |
| [spark](spark/) | [holman/spark](https://github.com/holman/spark) (bash, MIT, unmaintained since 2015) | rust rewrite: negatives and leading zeros, fair scaling, gaps for missing data, `--min/--max`, `--log`, multi-row charts, ascii fallback, color, `--width`, `--stats` |
| [rush](rush/) | [rednafi/rush](https://github.com/rednafi/rush) (Python, MIT, archived 2020) | zero dependencies, explicit `@task` dependencies with cycle detection, line-numbered parse errors, real exit codes, `--dry-run`, `--list --view`, utf-8 safe |
| [fixjson](fixjson/) | [SunGard-Labs/fix2json](https://github.com/SunGard-Labs/fix2json) (JavaScript, MIT, archived 2023) | zero dependencies, optional bundled or custom dictionaries, header/body/trailer output, repeating groups, `--validate`, JSON/JSONL/YAML, JSON back to FIX |
| [anon](anon/) | [intenthq/anon](https://github.com/intenthq/anon) (Go, MIT, archived 2019) | zero dependencies, header and named columns, pass-through columns, stable salted HMAC hashing, `mask`/`redact`/`ip`/`date` actions, line and column errors, deterministic sampling |
| [calc](calc/) | [alfredxing/calc](https://github.com/alfredxing/calc) (Go, MIT, archived 2023) | zero dependencies, real grammar (`-2^2`, right-assoc `^`), implicit multiplication, variables and `ans`/`@` history, more functions and `!`, column-numbered errors, works with pipes |
| [gohttpbench](gohttpbench/) | [parkghost/gohttpbench](https://github.com/parkghost/gohttpbench) (Go, MIT, archived 2023) | stdlib only, sub-millisecond latencies, partial report on Ctrl-C, `-d` duration mode, `--json`, categorised failures, real exit codes |
| [csscss](csscss/) | [zmoazeni/csscss](https://github.com/zmoazeni/csscss) (Ruby, MIT, archived 2018) | zero dependencies python rewrite, comment/`@charset`-safe parser, media-query scoping, same-selector matches, file and line locations, shorthand expansion, `--json`, `--fail-on-duplicates` |

## Layout

Every tool lives in its own folder with its own README (where it came from, why it died, what changed, how to install) and its own tests.

## Spin-outs

Twice a week, `.github/workflows/spinout.yml` gives one finished tool its own repo. A tool only moves out once it passes the gate in `.github/spinout/spinout.py`: it has a real entry point, it builds, at least 5 tests pass, `--help` runs, its README credits the original and shows it in use, and no repo already has its name or rebuilds the same original. The new repo keeps the folder's commit history and ships a `v0.1.0` pre-release with prebuilt binaries.
