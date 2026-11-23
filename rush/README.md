# rush

Run named bash snippets from a single `rushfile.yml`.

- **Original:** [rednafi/rush](https://github.com/rednafi/rush) by Redowan Delowar
- **License of original:** MIT. This is a from-scratch rewrite; no original code was copied.
- **Why it died:** archived in 2020 with open Python 3.6 and ASCII/UTF-8 problems, an unmerged dependency-graph PR, and a dependency on click and PyYAML.

## What's improved

- Zero dependencies: a small built-in parser, Python 3.9+.
- Explicit dependencies with `@task` lines, so a task named `ls` can't shadow a command.
- Cycle detection and errors that name the missing task and who needed it.
- Parse errors carry line numbers, and a missing `|` is caught (fixes #12, #13).
- The exit status of a failing task is the exit status of `rush` (fixes #14).
- `--dry-run`, `--list --view`, `--file`, and the rushfile is found in parent directories.
- Tasks run in the rushfile's directory with `bash -e`.

## Install

```
pip install .
```

## Rushfile

```yaml
build: |
    @test          # run `test` first
    make all
test: echo inline works too
#old: |            # leading # disables a task
    echo skipped
```

## Usage

```
rush --list [--view]      show tasks
rush build test           run tasks (dependencies first)
rush --all                run every enabled task
rush -n build             print run order only
rush --no-deps build      skip @dependencies
rush --ignore-errors ...  keep going after failures
rush --hide-outputs ...   discard output
rush -p                   print rushfile path
```

Exit status: 0 on success, the failing task's status on failure, 2 for usage or rushfile errors.

## Tests

```
python3 -m unittest discover -s tests -t .
```

Tracking issue: #10
