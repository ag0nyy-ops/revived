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
