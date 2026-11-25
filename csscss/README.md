# csscss

Find CSS rulesets that repeat the same declarations.

- **Original:** [zmoazeni/csscss](https://github.com/zmoazeni/csscss) by Zach Moazeni
- **License of original:** MIT. This is a from-scratch rewrite; no original code was copied.
- **Why it died:** archived in 2018. It was a Ruby gem tied to Ruby 1.9-era dependencies and the `sass`/`less` gems, with open bugs where comments and `@charset` leaked into selectors, media queries were compared as one pool, and identical selectors were never reported.

## What's improved

- Zero dependencies: a hand-written CSS parser, Python 3.9+.
- Comments, `@charset`, strings and `url(data:...;...)` are handled properly (fixes #48).
- Rulesets are only compared inside the same at-rule context, including nested `@media` and `@supports` (fixes #49).
- Rulesets with the same selector are reported, flagged `[same selector]`, and every result has file and line (fixes #50).
- `--json` output and `--fail-on-duplicates` for CI (fixes #51).
- `margin`, `padding` and `border-width/style/color` shorthands are expanded, so `margin: 0 auto` matches `margin-left: auto`. A shorthand counts once.
- `--ignore GLOB`, many files, and `-` for stdin.

## Install

```
pip install .
```

## Usage

```
csscss styles.css other.css          report rulesets sharing 3 or more declarations
csscss -n 5 -v styles.css            at least 5, and list the shared declarations
csscss --json styles.css             machine-readable
csscss -i '.vendor-*' styles.css     skip selectors matching a glob
csscss --fail-on-duplicates a.css    exit 1 if anything is found
cat a.css | csscss -                 read stdin
```

Example output:

```
{.contact .content} (a.css:12) and {article, #comments} (a.css:40) share 5 rules
{.card} (a.css:3) and {.card} (a.css:88) share 3 rules [same selector]
```

Exit status: 0 normally, 1 with `--fail-on-duplicates` when duplicates exist, 2 for usage, read or parse errors.

## Tests

```
python -m unittest discover -s tests -t .
```
