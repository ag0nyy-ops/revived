# eml2html

Rebuild of [dunnkers/eml-to-html](https://github.com/dunnkers/eml-to-html) by Jeroen Overschie (MIT, archived May 2024). This is a from-scratch rewrite; no original code was copied. The original MIT license and credit stand.

## Why the original died

It was a ~50 line script that stopped being maintained. It had real gaps: inputs that were "skipped" were still processed, plain-text emails produced empty files, every body was decoded as utf-8 whatever the charset said, and inline images (`cid:`) were lost.

## What's improved

- Skips and reports bad inputs, with a non-zero exit code
- Plain-text emails become escaped, wrapped HTML
- Honours the declared charset and never crashes on an unknown one
- Inline `cid:` images embedded as data URIs, so the output is a single file
- Optional header table (`--headers`)
- `--out-dir`, `--force`, `--stdout`, directory input and `-r`
- Standard library only, Python 3.9+, 15 tests

## Usage

```
pip install .
eml2html mail.eml                 # writes mail.html next to it
eml2html -r inbox/ -o html/       # whole tree into html/
eml2html --headers --stdout a.eml
python -m eml2html a.eml          # without installing
```

Existing output files are not overwritten unless you pass `--force`.

## Tests

```
python -m unittest discover -s tests
```
