# chglog

Generate a CHANGELOG from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) and git tags. One static Go binary, no dependencies, no config files.

```sh
chglog -o CHANGELOG.md          # full changelog
chglog --bump                   # next version: v1.4.0
chglog --next-tag auto --unreleased   # release notes for the version you're about to tag
chglog --latest                 # notes for the newest tag (paste into a GitHub release)
```

## Revived from

| | |
|---|---|
| original | [git-chglog/git-chglog](https://github.com/git-chglog/git-chglog) |
| license | MIT, © tsuyoshiwada |
| status | marked deprecated by its maintainers, who now point users to git-cliff |

This is a from-scratch rewrite, not a fork: no code was copied. It keeps the original's idea (a single binary that turns tags and commits into a changelog) and changes how you use it.

## What's different

- **Zero setup.** The original needed `git-chglog --init` to generate a config file and a template before the first run. chglog works out of the box with sensible Conventional Commits defaults; flags cover the rest.
- **Picks the next version for you.** `--bump` prints the next semver from unreleased commits (breaking → major, feat → minor, fix/perf/revert → patch, breaking before 1.0 → minor). `--next-tag auto` titles the unreleased section with it. The original's `--next-tag` was experimental and needed the version typed in by hand.
- **Commits land in the right release.** Releases are worked out by walking the commit graph: a commit belongs to the oldest tag that contains it. A branch written before `v2.0.0` but merged after it shows up under the next release, not under `v2.0.0`. All history is read with a single `git log`.
- **Reverts cancel out.** A commit and its revert in the same release are both dropped (`--keep-reverted` keeps them).
- **Full Conventional Commits 1.0 parsing.** `type(scope)!:` headers, multi-line `BREAKING CHANGE:` footers, git trailers, `Closes #12` and `(#12)` refs, git's own `Revert "..."` commits.
- **Monorepo friendly.** `--tag-prefix api/v` versions one package's tags, `--path api` keeps only commits touching that folder.
- **Machine-readable output.** `--format json` dumps releases and parsed commits for your own tooling.
- **Links for GitHub, GitLab and Bitbucket**, detected from the `origin` remote (ssh or https, credentials stripped) or set with `--repo-url`.
- **Tested.** Unit tests plus end-to-end tests that build real git repos with branches, annotated tags, merges and reverts.

## Install

Needs Go 1.22+ and git.

```sh
go install github.com/ag0nyy-ops/revived/chglog@latest
```

Or build from a clone:

```sh
cd chglog && go build -o chglog . && ./chglog -version
```

## Usage

Run it anywhere inside a git repo.

| flag | what it does |
|---|---|
| `-o FILE` | write to a file instead of stdout |
| `-C DIR` | run against another repo |
| `--format md\|json` | output format (default `md`) |
| `--bump` | print the next version and exit |
| `--next-tag TAG\|auto` | title unreleased changes as `TAG`, or the computed next version |
| `--unreleased` | only unreleased changes |
| `--latest` | only the newest tagged release |
| `--all` | also show docs, chore, refactor, test, build, ci, style and non-conventional commits |
| `--tag-prefix P` | only tags starting with `P` count as versions (default: optional `v`) |
| `--tag-filter RE` | only tags matching a regexp |
| `--skip-prereleases` | fold `-rc`/`-beta` tags into the following release |
| `--path P` | only commits touching `P` (repeatable) |
| `--merges` | include merge commits |
| `--keep-reverted` | keep commits reverted within the same release |
| `--repo-url URL` | base url for links (default: from `--remote`, which defaults to `origin`) |
| `--no-links` | plain hashes and issue numbers |
| `--title T` | document heading, `""` for none (default `Changelog`) |

Tags that aren't semantic versions (`latest`, `nightly`) are ignored.

### Example output

```md
## [v0.2.0](https://github.com/acme/widget/compare/v0.1.0...v0.2.0) (2026-01-02)

### ⚠ BREAKING CHANGES

- config.yml is now config.toml

### Features

- new config format ([a1b2c3d](https://github.com/acme/widget/commit/a1b2c3d...))

### Bug Fixes

- handle empty body ([#3](https://github.com/acme/widget/issues/3)) ([e4f5a6b](https://github.com/acme/widget/commit/e4f5a6b...))
```

### Cutting a release

```sh
next=$(chglog --bump)
chglog --next-tag "$next" -o CHANGELOG.md
git commit -am "chore(release): $next" && git tag "$next"
chglog --latest > notes.md   # for the release page
```

## Development

```sh
go test ./...
```

## License

MIT. See the [original project](https://github.com/git-chglog/git-chglog) for its own license and credits.
