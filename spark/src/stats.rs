//! A one-line summary to print under the sparkline.

use crate::parse::Point;
use std::fmt;

#[derive(Debug, Clone, PartialEq)]
pub struct Stats {
    pub min: f64,
    pub max: f64,
    pub mean: f64,
    pub last: f64,
    pub count: usize,
    pub missing: usize,
}

impl Stats {
    /// Summarizes the present values, or `None` if there are none.
    pub fn of(points: &[Point]) -> Option<Stats> {
        let values: Vec<f64> = points.iter().flatten().copied().collect();
        let last = *values.last()?;
        let sum: f64 = values.iter().sum();
        Some(Stats {
            min: values.iter().copied().fold(f64::INFINITY, f64::min),
            max: values.iter().copied().fold(f64::NEG_INFINITY, f64::max),
            mean: sum / values.len() as f64,
            last,
            count: values.len(),
            missing: points.len() - values.len(),
        })
    }
}

/// Prints whole numbers without a trailing `.0` and everything else with
/// at most three decimals, so the summary stays short.
fn num(v: f64) -> String {
    if v.fract() == 0.0 && v.abs() < 1e15 {
        format!("{}", v as i64)
    } else {
        let s = format!("{v:.3}");
        s.trim_end_matches('0').trim_end_matches('.').to_string()
    }
}

impl fmt::Display for Stats {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "min {}  max {}  avg {}  last {}  n {}",
            num(self.min),
            num(self.max),
            num(self.mean),
            num(self.last),
            self.count
        )?;
        if self.missing > 0 {
            write!(f, "  missing {}", self.missing)?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn summarizes_present_values() {
        let s = Stats::of(&[Some(1.0), None, Some(5.0), Some(22.0), Some(13.0)]).unwrap();
        assert_eq!(s.min, 1.0);
        assert_eq!(s.max, 22.0);
        assert_eq!(s.mean, 10.25);
        assert_eq!(s.last, 13.0);
        assert_eq!(s.missing, 1);
        assert_eq!(
            s.to_string(),
            "min 1  max 22  avg 10.25  last 13  n 4  missing 1"
        );
    }

    #[test]
    fn rounds_long_decimals() {
        let s = Stats::of(&[Some(1.0), Some(2.0), Some(2.0)]).unwrap();
        assert_eq!(s.to_string(), "min 1  max 2  avg 1.667  last 2  n 3");
    }

    #[test]
    fn nothing_to_summarize() {
        assert_eq!(Stats::of(&[None]), None);
        assert_eq!(Stats::of(&[]), None);
    }
}
