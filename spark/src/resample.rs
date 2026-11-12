//! Squashing a long series into a fixed number of columns.

use crate::parse::Point;

/// Averages `points` into at most `width` buckets.
///
/// Buckets are as even as integer math allows: a 10-point series into 4
/// columns gets sizes 2, 3, 2, 3. Gaps are skipped when averaging, and a
/// bucket that is entirely gaps stays a gap. Shorter series are returned
/// untouched, since stretching would invent data.
pub fn resample(points: &[Point], width: usize) -> Vec<Point> {
    if width == 0 || points.len() <= width {
        return points.to_vec();
    }
    let n = points.len();
    (0..width)
        .map(|i| {
            let start = i * n / width;
            let end = (i + 1) * n / width;
            let (sum, count) = points[start..end]
                .iter()
                .flatten()
                .fold((0.0, 0usize), |(s, c), v| (s + v, c + 1));
            (count > 0).then(|| sum / count as f64)
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn short_series_unchanged() {
        let p = vec![Some(1.0), None, Some(2.0)];
        assert_eq!(resample(&p, 10), p);
        assert_eq!(resample(&p, 0), p);
    }

    #[test]
    fn averages_into_buckets() {
        let p: Vec<Point> = (1..=8).map(|v| Some(v as f64)).collect();
        assert_eq!(
            resample(&p, 4),
            vec![Some(1.5), Some(3.5), Some(5.5), Some(7.5)]
        );
    }

    #[test]
    fn uneven_buckets_cover_everything() {
        let p: Vec<Point> = (0..10).map(|v| Some(v as f64)).collect();
        let out = resample(&p, 4);
        assert_eq!(out.len(), 4);
        assert_eq!(out[0], Some(0.5)); // 0,1
        assert_eq!(out[3], Some(8.0)); // 7,8,9
    }

    #[test]
    fn gaps_skipped_or_kept() {
        let p = vec![Some(2.0), None, None, None];
        assert_eq!(resample(&p, 2), vec![Some(2.0), None]);
    }
}
