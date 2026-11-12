# spark

Sparklines for your shell. Give it numbers, get `▁▂▃▅▇` back. One small Rust binary, no dependencies.

```sh
spark 1 5 22 13 53                    # ▁▁▄▂█
seq 0 9 | spark                       # ▁▁▂▃▄▅▆▇██
spark -5 0 5                          # ▁▅█   negatives just work
spark 1 x 3 nan 9                     # ▁ ▃ █ missing values are gaps
spark --stats 12 15 9 31 44 40 x 28 19
# ▁▂▁▆██ ▅▃
# min 9  max 44  avg 24.75  last 19  n 8  missing 1
```

A taller chart with `--rows`:

```
$ seq 1 40 | awk '{print int(50+40*sin($1/4))}' | spark -r 3
 ▂▅▇████▆▄▁               ▂▅▆████▆▄▂
▇██████████▆▃          ▁▄▇██████████▆▃▁
██████████████▆▃▂▁▁▁▂▄▆████████████████▆
```

## Revived from

| | |
|---|---|
| original | [holman/spark](https://github.com/holman/spark) |
| license | MIT, © Zach Holman |
| status | single bash script; last real change in 2015, open bugs and pull requests unanswered since |

This is a from-scratch rewrite, not a fork: no code was copied. It keeps the original's idea and its default look (eight block characters, numbers from args or stdin) and fixes what people kept reporting.

## What's different

- **Negative numbers, decimals, leading zeros.** `-5` is a value, not a flag. `007` is seven, not an octal error. `1e3` and `0.25` work. Upstream issues #108 and #93.
- **Piping works everywhere.** Numbers come from args or stdin, separated by any mix of spaces, commas, semicolons and newlines. Upstream #106.
- **A fair scale.** Each of the eight bars covers the same slice of the range, so values near the top get the tall bar, not only the exact maximum. Flat input draws mid-height instead of dividing by zero. Upstream #101.
- **Missing data.** `x`, `_`, `?`, `nan`, `n/a`, `null` and empty CSV fields (`1,,3`) render as gaps. Upstream PR #97.
- **Fixed scale.** `--min` and `--max` pin the scale and clamp outliers, handy for percentages. Upstream added this and then reverted it.
- **Log scale.** `--log` for data that spans orders of magnitude. Upstream #102 and PR #99.
- **Taller charts.** `--rows N` draws N lines with partial blocks on top, giving N×8 distinct heights. Upstream #110 and PR #103.
- **No mojibake.** On a locale that isn't UTF-8 (like `C`), spark falls back to `_.-~=+*#`. `--ascii` and `--unicode` force either. Upstream #98.
- **Color.** `--color` shades bars green, yellow and red by level when stdout is a terminal (`--color=always` to force). `NO_COLOR` is respected. Upstream PR #94.
- **Long series.** `--width N` averages the data into N columns.
- **Real errors.** Bad input names the offending value and its position on stderr and exits 1. Bad options exit 2. Run with nothing on a terminal and you get help instead of a hang.

One deliberate change: because the scale is fairer, a few values land one bar higher than they did upstream (the classic `1 5 22 13 53` example shows `▄` for 22 where the original showed `▃`).

## Usage

```
spark [options] [--] [numbers...]

      --min N            fix the bottom of the scale (values below clamp)
      --max N            fix the top of the scale (values above clamp)
      --log              log10 scale, for data spanning orders of magnitude
  -r, --rows N           draw a chart N lines tall (default 1)
  -w, --width N          average the data into at most N columns
      --ascii            use _.-~=+*# instead of block characters
      --unicode          always use block characters
      --color[=WHEN]     color by level: auto, always or never
  -s, --stats            print min, max, avg and last under the chart
  -h, --help             show help
  -V, --version          show the version
```

Some ideas:

```sh
# commits per day over the last two weeks
for d in $(seq 13 -1 0); do git log --oneline --since="$d days ago 00:00" --until="$d days ago 23:59" | wc -l; done | spark

# disk usage of each folder, on a log scale
du -s */ | cut -f1 | spark --log

# cpu load samples, always 0-100
spark --min 0 --max 100 --color 12 40 85 97 60 22
```

## Install

```sh
cargo install --path spark
```

or build it and copy the binary anywhere on your `PATH`:

```sh
cd spark && cargo build --release && cp target/release/spark ~/.local/bin/
```

Needs Rust 1.70 or newer. No crates are pulled in.

## Tests

```sh
cd spark && cargo test
```

Unit tests cover each stage (parse, scale, resample, render, stats, option parsing) and `tests/cli.rs` runs the built binary end to end.
