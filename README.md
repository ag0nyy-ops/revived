# revived

Rebuilt and improved versions of abandoned open-source tools.

Each folder is a from-scratch rewrite of a tool that was archived, deprecated or left unmaintained. Only permissively licensed originals (MIT, Apache-2.0, BSD, ISC, Unlicense) are picked, every rebuild credits the original, and no original code is copied. The goal for each one: modern runtime, known issues fixed, simpler to use, and tests that actually run.

## Tools

| tool | original | what's new |
|---|---|---|
| [chglog](chglog/) | [git-chglog/git-chglog](https://github.com/git-chglog/git-chglog) (Go, MIT, deprecated) | zero-config, auto next version (`--bump`), graph-correct release grouping, revert cancelling, monorepo tags and paths, JSON output, zero deps |

## Layout

Every tool lives in its own folder with its own README (where it came from, why it died, what changed, how to install) and its own tests.
