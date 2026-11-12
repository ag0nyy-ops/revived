//! Command-line options and the glue that runs the pipeline.

use crate::parse::{is_missing, parse_number, parse_series};
use crate::render::{self, Charset};
use crate::resample::resample;
use crate::scale::{normalize, Scale};
use crate::stats::Stats;

pub const USAGE: &str = "usage: spark [options] [--] [numbers...]

Prints a sparkline for the numbers given as arguments, or read from stdin
when there are none. Numbers can be separated by spaces, commas,
semicolons or newlines. x, _, nan and empty csv fields are drawn as gaps.

  spark 1 5 22 13 53          ▁▁▄▂█
  seq 0 9 | spark              ▁▁▂▃▄▅▆▇██
  spark -3 -1 0 2              ▁▄▅█

options:
      --min N            fix the bottom of the scale (values below clamp)
      --max N            fix the top of the scale (values above clamp)
      --log              log10 scale, for data spanning orders of magnitude
  -r, --rows N           draw a chart N lines tall (default 1)
  -w, --width N          average the data into at most N columns
      --ascii            use _.-~=+*# instead of block characters
      --unicode          always use block characters
      --color[=WHEN]     color by level: auto, always or never (default auto
                         when the flag is given, never otherwise)
  -s, --stats            print min, max, avg and last under the chart
  -h, --help             show this help
  -V, --version          show the version

NO_COLOR turns off --color=auto. The locale (LC_ALL, LC_CTYPE, LANG)
picks ascii when it isn't UTF-8, unless --unicode is given.";

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum ColorWhen {
    Auto,
    Always,
    #[default]
    Never,
}

#[derive(Debug, Default, PartialEq)]
pub struct Options {
    pub scale_min: Option<f64>,
    pub scale_max: Option<f64>,
    pub log: bool,
    pub rows: usize,
    pub width: Option<usize>,
    pub charset: Option<Charset>,
    pub color: ColorWhen,
    pub stats: bool,
    pub help: bool,
    pub version: bool,
    /// Numbers given on the command line, joined back into text.
    pub values: Vec<String>,
}

fn take_value<'a, I: Iterator<Item = &'a String>>(
    name: &str,
    inline: Option<&'a str>,
    rest: &mut I,
) -> Result<&'a str, String> {
    match inline {
        Some(v) => Ok(v),
        None => rest
            .next()
            .map(String::as_str)
            .ok_or_else(|| format!("{name} needs a value")),
    }
}

fn number(name: &str, v: &str) -> Result<f64, String> {
    parse_number(v).ok_or_else(|| format!("{name} expects a number, got {v:?}"))
}

fn count(name: &str, v: &str) -> Result<usize, String> {
    match v.parse::<usize>() {
        Ok(n) if n > 0 => Ok(n),
        _ => Err(format!(
            "{name} expects a whole number above zero, got {v:?}"
        )),
    }
}

impl Options {
    /// Parses arguments (without the program name).
    ///
    /// Anything that reads as a number or a missing marker is data, even
    /// when it starts with a dash, so `spark -5 3 10` just works. `--`
    /// still ends option parsing for anyone who wants to be explicit.
    pub fn parse(args: &[String]) -> Result<Options, String> {
        let mut o = Options {
            rows: 1,
            ..Options::default()
        };
        let mut it = args.iter();
        while let Some(arg) = it.next() {
            if arg == "--" {
                o.values.extend(it.by_ref().cloned());
                break;
            }
            if parse_number(arg).is_some() || is_missing(arg) || !arg.starts_with('-') {
                o.values.push(arg.clone());
                continue;
            }
            let (name, inline) = match arg.split_once('=') {
                Some((n, v)) if n.starts_with("--") => (n, Some(v)),
                _ => (arg.as_str(), None),
            };
            match name {
                "-h" | "--help" => o.help = true,
                "-V" | "--version" => o.version = true,
                "--min" => o.scale_min = Some(number(name, take_value(name, inline, &mut it)?)?),
                "--max" => o.scale_max = Some(number(name, take_value(name, inline, &mut it)?)?),
                "--log" => o.log = true,
                "-r" | "--rows" => o.rows = count(name, take_value(name, inline, &mut it)?)?,
                "-w" | "--width" => {
                    o.width = Some(count(name, take_value(name, inline, &mut it)?)?)
                }
                "--ascii" => o.charset = Some(Charset::Ascii),
                "--unicode" => o.charset = Some(Charset::Unicode),
                "--color" | "--colour" => {
                    o.color = match inline.unwrap_or("auto") {
                        "auto" => ColorWhen::Auto,
                        "always" => ColorWhen::Always,
                        "never" => ColorWhen::Never,
                        other => {
                            return Err(format!(
                                "--color expects auto, always or never, got {other:?}"
                            ))
                        }
                    }
                }
                "-s" | "--stats" => o.stats = true,
                _ => return Err(format!("unknown option {arg:?} (see spark --help)")),
            }
        }
        Ok(o)
    }
}

/// What the environment looks like, so `render` stays testable.
pub struct Env {
    pub charset: Charset,
    pub stdout_is_tty: bool,
    pub no_color: bool,
}

/// Runs the whole pipeline on `input` and returns the lines to print.
pub fn render(opts: &Options, input: &str, env: &Env) -> Result<Vec<String>, String> {
    let points = parse_series(input).map_err(|e| e.to_string())?;
    if points.iter().all(Option::is_none) {
        return Err("no numbers to draw".into());
    }
    let shown = match opts.width {
        Some(w) => resample(&points, w),
        None => points.clone(),
    };
    let scale = Scale {
        min: opts.scale_min,
        max: opts.scale_max,
        log: opts.log,
    };
    let fracs = normalize(&shown, &scale).map_err(|e| e.to_string())?;
    let charset = opts.charset.unwrap_or(env.charset);
    let mut lines = render::rows(&fracs, opts.rows, charset);
    let color = match opts.color {
        ColorWhen::Always => true,
        ColorWhen::Never => false,
        ColorWhen::Auto => env.stdout_is_tty && !env.no_color,
    };
    if color {
        lines = render::paint(&fracs, &lines);
    }
    if opts.stats {
        // Stats describe the real data, not the resampled columns.
        if let Some(s) = Stats::of(&points) {
            lines.push(s.to_string());
        }
    }
    Ok(lines)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn args(s: &str) -> Vec<String> {
        s.split_whitespace().map(String::from).collect()
    }

    fn env() -> Env {
        Env {
            charset: Charset::Unicode,
            stdout_is_tty: false,
            no_color: false,
        }
    }

    #[test]
    fn negative_numbers_are_values_not_flags() {
        let o = Options::parse(&args("-5 3 -0.5 10")).unwrap();
        assert_eq!(o.values, args("-5 3 -0.5 10"));
    }

    #[test]
    fn double_dash_ends_options() {
        let o = Options::parse(&args("--stats -- --1")).unwrap();
        assert!(o.stats);
        assert_eq!(o.values, args("--1"));
    }

    #[test]
    fn options_with_separate_and_inline_values() {
        let o = Options::parse(&args("--min 0 --max=100 -r 3 --width=40 --log")).unwrap();
        assert_eq!(o.scale_min, Some(0.0));
        assert_eq!(o.scale_max, Some(100.0));
        assert_eq!(o.rows, 3);
        assert_eq!(o.width, Some(40));
        assert!(o.log);
    }

    #[test]
    fn min_accepts_negative_value() {
        let o = Options::parse(&args("--min -10 1 2")).unwrap();
        assert_eq!(o.scale_min, Some(-10.0));
        assert_eq!(o.values, args("1 2"));
    }

    #[test]
    fn bad_options_are_reported() {
        assert!(Options::parse(&args("--bogus")).is_err());
        assert!(Options::parse(&args("--rows 0")).is_err());
        assert!(Options::parse(&args("--min")).is_err());
        assert!(Options::parse(&args("--color=rainbow")).is_err());
    }

    #[test]
    fn classic_example() {
        let o = Options::parse(&[]).unwrap();
        let out = render(&o, "1 5 22 13 53", &env()).unwrap();
        assert_eq!(out, vec!["▁▁▄▂█"]);
    }

    #[test]
    fn color_auto_needs_a_tty_and_no_no_color() {
        let o = Options::parse(&args("--color")).unwrap();
        let plain = render(&o, "1 2", &env()).unwrap();
        assert!(!plain[0].contains('\x1b'));
        let tty = Env {
            stdout_is_tty: true,
            ..env()
        };
        assert!(render(&o, "1 2", &tty).unwrap()[0].contains('\x1b'));
        let no = Env {
            stdout_is_tty: true,
            no_color: true,
            ..env()
        };
        assert!(!render(&o, "1 2", &no).unwrap()[0].contains('\x1b'));
    }

    #[test]
    fn empty_input_is_an_error() {
        let o = Options::parse(&[]).unwrap();
        assert!(render(&o, "  \n", &env()).is_err());
        assert!(render(&o, "x x", &env()).is_err());
    }
}
