package expr

import (
	"math"
	"strconv"
)

// Format renders a result the way a calculator would: integers without a
// fraction, otherwise the shortest representation that round-trips.
func Format(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	case v == math.Trunc(v) && math.Abs(v) < 1e15:
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}
