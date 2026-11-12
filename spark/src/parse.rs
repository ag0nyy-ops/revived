//! Turning raw text into a series of values.
//!
//! Input can be separated by any mix of whitespace, commas and semicolons,
//! so `1 2 3`, `1,2,3` and one-number-per-line output from other tools all
//! work the same way.

use std::fmt;

/// A single point in the series. `None` marks a missing value.
pub type Point = Option<f64>;

#[derive(Debug, PartialEq)]
pub struct ParseError {
    pub token: String,
    pub position: usize,
}

impl fmt::Display for ParseError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "value {} is not a number: {:?}",
            self.position + 1,
            self.token
        )
    }
}

impl std::error::Error for ParseError {}

fn is_separator(c: char) -> bool {
    c.is_whitespace() || c == ',' || c == ';'
}

/// Parses one token as a number.
///
/// Rust's float parser already copes with negatives, leading zeros
/// (`007`), decimals and scientific notation, which is exactly what the
/// original bash arithmetic choked on. A leading `+` is allowed too.
pub fn parse_number(token: &str) -> Option<f64> {
    let t = token.trim();
    if t.is_empty() {
        return None;
    }
    // f64::from_str accepts "inf" and "nan"; those are not useful data.
    let lower = t.to_ascii_lowercase();
    if lower.contains("inf") || lower.contains("nan") {
        return None;
    }
    t.parse::<f64>().ok().filter(|v| v.is_finite())
}

/// Splits text into tokens and parses each one.
pub fn parse_series(text: &str) -> Result<Vec<Point>, ParseError> {
    text.split(is_separator)
        .filter(|t| !t.is_empty())
        .enumerate()
        .map(|(i, tok)| match parse_number(tok) {
            Some(v) => Ok(Some(v)),
            None => Err(ParseError {
                token: tok.to_string(),
                position: i,
            }),
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn vals(s: &str) -> Vec<f64> {
        parse_series(s)
            .unwrap()
            .into_iter()
            .map(|p| p.unwrap())
            .collect()
    }

    #[test]
    fn mixed_separators() {
        assert_eq!(vals("1 2,3;4\n5\t6"), vec![1.0, 2.0, 3.0, 4.0, 5.0, 6.0]);
        assert_eq!(vals("1, 2 ,, 3"), vec![1.0, 2.0, 3.0]);
    }

    #[test]
    fn negatives_and_decimals() {
        assert_eq!(vals("-5 3.5 -0.25 +2"), vec![-5.0, 3.5, -0.25, 2.0]);
    }

    #[test]
    fn leading_zeros_are_decimal_not_octal() {
        assert_eq!(vals("007 010 0.50"), vec![7.0, 10.0, 0.5]);
    }

    #[test]
    fn scientific_notation() {
        assert_eq!(vals("1e3 2.5E-1"), vec![1000.0, 0.25]);
    }

    #[test]
    fn rejects_garbage_with_position() {
        let err = parse_series("1 2 banana").unwrap_err();
        assert_eq!(err.token, "banana");
        assert_eq!(err.position, 2);
        assert_eq!(err.to_string(), "value 3 is not a number: \"banana\"");
    }

    #[test]
    fn rejects_inf() {
        assert!(parse_series("1 inf").is_err());
    }
}
