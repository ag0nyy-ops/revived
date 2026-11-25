# calc

A small command-line calculator, rebuilt in Go with no dependencies.

- **Original:** [alfredxing/calc](https://github.com/alfredxing/calc) by Alfred Xing, MIT license
- **Rebuilt from scratch.** No original source was copied; only the idea and the documented behaviour (functions, `@` history) were used as a reference. The original's MIT license and authorship are credited here.

## Why the original died

The repository was archived in 2023. It relied on `golang.org/x/crypto/ssh/terminal` (deprecated years ago), had no go modules, needed a real terminal for shell mode, handled unary minus and exponents with token rewriting, and reported bad input as `NaN`.

## What's improved

- Zero dependencies, Go modules
- Real grammar: `-2^2 = -4`, `2^-2 = 0.25`, `2^3^2 = 512`
- Implicit multiplication: `3pi`, `2(3+4)`, `(1+2)(3+4)`, `x y`
- Variables (`x = 4`), `ans`, and `@` / `@@` history as in the original
- More functions: sinh/cosh/tanh, floor, ceil, round, trunc, cbrt, exp, log2, pow, mod, atan2, hypot, min, max, sum, avg; plus `!` factorial, `tau`, `phi`, `1_000` and `1e3` literals
- Errors with a column and a non-zero exit code (`1/0`, `ln(-1)`, `(2`, unknown names)
- Works with pipes: `echo '1+1' | calc`
- Tests for the parser and the CLI

## Usage

```
$ go build -o calc .
$ calc '3(5/(3-4))'
-15
$ calc 3pi^2
29.608813203268074
$ printf 'x = 4\nx!\n@ + 1\n' | calc
4
24
25
$ calc '1 +'
calc: column 4: unexpected end of expression
```

With no arguments and a terminal on stdin, calc prompts with `> `; `exit` or Ctrl-D quits. Lines starting with `#` are ignored.

Run the tests with `go test ./...`.
