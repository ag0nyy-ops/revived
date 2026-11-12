//! Drawing levels as text.

use crate::scale::quantize;

/// The eight block heights used for a one-row sparkline.
pub const UNICODE_TICKS: [char; 8] = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];

/// An ascii ramp for terminals that can't show block characters.
pub const ASCII_TICKS: [char; 8] = ['_', '.', '-', '~', '=', '+', '*', '#'];

/// What gaps (missing values) are drawn as.
pub const GAP: char = ' ';

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Charset {
    Unicode,
    Ascii,
}

impl Charset {
    pub fn ticks(self) -> &'static [char; 8] {
        match self {
            Charset::Unicode => &UNICODE_TICKS,
            Charset::Ascii => &ASCII_TICKS,
        }
    }

    /// Picks a charset from the locale, the way most CLI tools do.
    ///
    /// The first of `LC_ALL`, `LC_CTYPE` and `LANG` that is set decides.
    /// A locale that isn't UTF-8 (like `C` or `en_US.ISO-8859-1`) gets the
    /// ascii ramp, which is what fixes the mojibake people saw upstream.
    /// With nothing set at all we assume a modern UTF-8 terminal.
    pub fn from_locale(get: impl Fn(&str) -> Option<String>) -> Charset {
        let locale = ["LC_ALL", "LC_CTYPE", "LANG"]
            .iter()
            .filter_map(|k| get(k))
            .find(|v| !v.is_empty());
        match locale {
            None => Charset::Unicode,
            Some(v) => {
                let v = v.to_ascii_lowercase();
                if v.contains("utf-8") || v.contains("utf8") {
                    Charset::Unicode
                } else {
                    Charset::Ascii
                }
            }
        }
    }
}

/// Renders normalized values as a single line.
pub fn line(fracs: &[Option<f64>], charset: Charset) -> String {
    let ticks = charset.ticks();
    fracs
        .iter()
        .map(|f| match f {
            Some(f) => ticks[quantize(*f, ticks.len())],
            None => GAP,
        })
        .collect()
}

/// Renders normalized values as a chart `rows` lines tall.
///
/// Each column gets a height in eighths of a cell. Full cells are drawn
/// with the tallest tick and the top cell of each column gets the partial
/// block, so a 3-row chart has 24 distinct heights instead of 8.
/// With `rows == 1` this is identical to [`line`].
pub fn rows(fracs: &[Option<f64>], rows: usize, charset: Charset) -> Vec<String> {
    let rows = rows.max(1);
    let ticks = charset.ticks();
    let steps = ticks.len();
    let heights: Vec<Option<usize>> = fracs
        .iter()
        .map(|f| f.map(|f| quantize(f, rows * steps) + 1))
        .collect();
    (0..rows)
        .map(|r| {
            let base = (rows - 1 - r) * steps;
            heights
                .iter()
                .map(|h| match h {
                    Some(h) if *h > base => ticks[(h - base).min(steps) - 1],
                    _ => GAP,
                })
                .collect()
        })
        .collect()
}

/// ANSI color for a level: green for the bottom third, yellow for the
/// middle, red for the top. Handy for load, latency or error counts.
pub fn color_for(frac: f64) -> &'static str {
    match quantize(frac, 3) {
        0 => "\x1b[32m",
        1 => "\x1b[33m",
        _ => "\x1b[31m",
    }
}

const RESET: &str = "\x1b[0m";

/// Wraps every drawn cell of `lines` in the color for its column.
///
/// Gaps are left alone so they don't carry stray escape codes.
pub fn paint(fracs: &[Option<f64>], lines: &[String]) -> Vec<String> {
    lines
        .iter()
        .map(|line| {
            let mut out = String::new();
            for (c, f) in line.chars().zip(fracs) {
                match f {
                    Some(f) if c != GAP => {
                        out.push_str(color_for(*f));
                        out.push(c);
                        out.push_str(RESET);
                    }
                    _ => out.push(c),
                }
            }
            out
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn env<'a>(pairs: &'a [(&'a str, &'a str)]) -> impl Fn(&str) -> Option<String> + 'a {
        move |k| {
            pairs
                .iter()
                .find(|(name, _)| *name == k)
                .map(|(_, v)| v.to_string())
        }
    }

    #[test]
    fn draws_unicode_line() {
        let fracs: Vec<Option<f64>> = (0..8).map(|i| Some(i as f64 / 7.0)).collect();
        assert_eq!(line(&fracs, Charset::Unicode), "▁▂▃▄▅▆▇█");
    }

    #[test]
    fn draws_ascii_line_with_gap() {
        assert_eq!(line(&[Some(0.0), None, Some(1.0)], Charset::Ascii), "_ #");
    }

    #[test]
    fn one_row_matches_line() {
        let fracs = [Some(0.0), Some(0.3), None, Some(0.7), Some(1.0)];
        assert_eq!(
            rows(&fracs, 1, Charset::Unicode),
            vec![line(&fracs, Charset::Unicode)]
        );
    }

    #[test]
    fn two_rows_stack_blocks() {
        let fracs = [Some(0.0), Some(0.5), Some(1.0), None];
        assert_eq!(rows(&fracs, 2, Charset::Unicode), vec![" ▁█ ", "▁██ "]);
    }

    #[test]
    fn three_rows_have_finer_steps() {
        // 24 heights: 0.5 lands on height 13, a full bottom cell plus five
        // eighths of the middle one.
        let out = rows(&[Some(0.5)], 3, Charset::Unicode);
        assert_eq!(out, vec![" ", "▅", "█"]);
    }

    #[test]
    fn paints_by_level_and_skips_gaps() {
        let fracs = [Some(0.0), None, Some(0.5), Some(1.0)];
        let out = paint(&fracs, &[line(&fracs, Charset::Ascii)]);
        assert_eq!(out[0], "\x1b[32m_\x1b[0m \x1b[33m=\x1b[0m\x1b[31m#\x1b[0m");
    }

    #[test]
    fn paint_leaves_empty_cells_of_tall_charts_plain() {
        let fracs = [Some(0.0), Some(1.0)];
        let out = paint(&fracs, &rows(&fracs, 2, Charset::Unicode));
        assert_eq!(out[0], " \x1b[31m█\x1b[0m");
    }

    #[test]
    fn locale_detection() {
        assert_eq!(Charset::from_locale(env(&[])), Charset::Unicode);
        assert_eq!(
            Charset::from_locale(env(&[("LANG", "en_US.UTF-8")])),
            Charset::Unicode
        );
        assert_eq!(Charset::from_locale(env(&[("LANG", "C")])), Charset::Ascii);
        // LC_ALL wins over LANG.
        assert_eq!(
            Charset::from_locale(env(&[("LC_ALL", "POSIX"), ("LANG", "en_US.utf8")])),
            Charset::Ascii
        );
        // Empty values are skipped.
        assert_eq!(
            Charset::from_locale(env(&[("LC_ALL", ""), ("LANG", "de_DE.utf8")])),
            Charset::Unicode
        );
    }
}
