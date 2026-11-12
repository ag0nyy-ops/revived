//! Mapping values onto a 0..=1 range and then onto discrete levels.

use crate::parse::Point;
use std::fmt;

/// How values are mapped before they're drawn.
#[derive(Debug, Clone, Default)]
pub struct Scale {
    /// Fixed bottom of the scale. Values below it are clamped.
    pub min: Option<f64>,
    /// Fixed top of the scale. Values above it are clamped.
    pub max: Option<f64>,
}

#[derive(Debug, PartialEq)]
pub enum ScaleError {
    MinAboveMax { min: f64, max: f64 },
}

impl fmt::Display for ScaleError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ScaleError::MinAboveMax { min, max } => {
                write!(f, "--min ({min}) is greater than --max ({max})")
            }
        }
    }
}

impl std::error::Error for ScaleError {}

/// Returns the lowest and highest present value, ignoring gaps.
pub fn bounds(points: &[Point]) -> Option<(f64, f64)> {
    points.iter().flatten().fold(None, |acc, &v| match acc {
        None => Some((v, v)),
        Some((lo, hi)) => Some((lo.min(v), hi.max(v))),
    })
}

/// Maps every present value to a fraction in `0.0..=1.0`.
///
/// When every value is the same there's no range to spread over, so
/// everything lands in the middle rather than dividing by zero.
pub fn normalize(points: &[Point], scale: &Scale) -> Result<Vec<Option<f64>>, ScaleError> {
    let (data_lo, data_hi) = match bounds(points) {
        Some(b) => b,
        None => return Ok(points.iter().map(|_| None).collect()),
    };
    let lo = scale.min.unwrap_or(data_lo);
    let hi = scale.max.unwrap_or(data_hi);
    if lo > hi {
        return Err(ScaleError::MinAboveMax { min: lo, max: hi });
    }
    let range = hi - lo;
    Ok(points
        .iter()
        .map(|p| {
            p.map(|v| {
                if range == 0.0 {
                    0.5
                } else {
                    ((v - lo) / range).clamp(0.0, 1.0)
                }
            })
        })
        .collect())
}

/// Turns a fraction into one of `levels` equal-width buckets.
///
/// Each bucket covers the same slice of the range, so the top bar isn't
/// reserved for the single largest value like it was upstream.
pub fn quantize(frac: f64, levels: usize) -> usize {
    debug_assert!(levels > 0);
    ((frac * levels as f64).floor() as usize).min(levels - 1)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn some(v: &[f64]) -> Vec<Point> {
        v.iter().copied().map(Some).collect()
    }

    fn levels(v: &[f64], scale: &Scale) -> Vec<usize> {
        normalize(&some(v), scale)
            .unwrap()
            .into_iter()
            .map(|f| quantize(f.unwrap(), 8))
            .collect()
    }

    #[test]
    fn spreads_evenly_over_eight_levels() {
        let v: Vec<f64> = (0..8).map(f64::from).collect();
        assert_eq!(levels(&v, &Scale::default()), (0..8).collect::<Vec<_>>());
    }

    #[test]
    fn near_max_values_share_the_top_bar() {
        // 95 and 100 are in the same top slice of 0..100, so both get
        // the tallest bar instead of only the exact maximum.
        assert_eq!(levels(&[0.0, 95.0, 100.0], &Scale::default()), vec![0, 7, 7]);
    }

    #[test]
    fn handles_negative_ranges() {
        assert_eq!(levels(&[-10.0, 0.0, 10.0], &Scale::default()), vec![0, 4, 7]);
    }

    #[test]
    fn flat_input_sits_in_the_middle() {
        assert_eq!(levels(&[3.0, 3.0, 3.0], &Scale::default()), vec![4, 4, 4]);
    }

    #[test]
    fn fixed_min_max_clamps() {
        let s = Scale {
            min: Some(0.0),
            max: Some(10.0),
        };
        assert_eq!(levels(&[-5.0, 5.0, 50.0], &s), vec![0, 4, 7]);
    }

    #[test]
    fn min_above_max_is_an_error() {
        let s = Scale {
            min: Some(10.0),
            max: Some(1.0),
        };
        assert!(normalize(&some(&[1.0]), &s).is_err());
    }

    #[test]
    fn gaps_pass_through() {
        let n = normalize(&[Some(1.0), None, Some(3.0)], &Scale::default()).unwrap();
        assert_eq!(n, vec![Some(0.0), None, Some(1.0)]);
    }

    #[test]
    fn all_gaps_is_not_an_error() {
        let n = normalize(&[None, None], &Scale::default()).unwrap();
        assert_eq!(n, vec![None, None]);
    }
}
