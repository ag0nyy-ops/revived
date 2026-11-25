package expr

import (
	"math"
)

type function struct {
	min, max int // max < 0 means variadic
	fn       func(a []float64) (float64, error)
}

func f1(f func(float64) float64) function {
	return function{1, 1, func(a []float64) (float64, error) { return f(a[0]), nil }}
}

func f2(f func(a, b float64) float64) function {
	return function{2, 2, func(a []float64) (float64, error) { return f(a[0], a[1]), nil }}
}

var functions = map[string]function{
	"sin": f1(math.Sin), "cos": f1(math.Cos), "tan": f1(math.Tan),
	"cot":  f1(func(x float64) float64 { return 1 / math.Tan(x) }),
	"sec":  f1(func(x float64) float64 { return 1 / math.Cos(x) }),
	"csc":  f1(func(x float64) float64 { return 1 / math.Sin(x) }),
	"asin": f1(math.Asin), "acos": f1(math.Acos), "atan": f1(math.Atan),
	"acot": f1(func(x float64) float64 { return math.Atan(1 / x) }),
	"asec": f1(func(x float64) float64 { return math.Acos(1 / x) }),
	"acsc": f1(func(x float64) float64 { return math.Asin(1 / x) }),
	"sinh": f1(math.Sinh), "cosh": f1(math.Cosh), "tanh": f1(math.Tanh),
	"sqrt": f1(math.Sqrt), "cbrt": f1(math.Cbrt),
	"log": f1(math.Log10), "lg": f1(math.Log2), "log2": f1(math.Log2),
	"ln": f1(math.Log), "exp": f1(math.Exp),
	"abs": f1(math.Abs), "floor": f1(math.Floor), "ceil": f1(math.Ceil),
	"trunc": f1(math.Trunc),
	"round": f1(math.Round),
	"pow":   f2(math.Pow), "atan2": f2(math.Atan2), "hypot": f2(math.Hypot),
	"mod": f2(math.Mod),
	"min": {1, -1, func(a []float64) (float64, error) {
		m := a[0]
		for _, v := range a[1:] {
			m = math.Min(m, v)
		}
		return m, nil
	}},
	"max": {1, -1, func(a []float64) (float64, error) {
		m := a[0]
		for _, v := range a[1:] {
			m = math.Max(m, v)
		}
		return m, nil
	}},
	"sum": {1, -1, func(a []float64) (float64, error) {
		s := 0.0
		for _, v := range a {
			s += v
		}
		return s, nil
	}},
	"avg": {1, -1, func(a []float64) (float64, error) {
		s := 0.0
		for _, v := range a {
			s += v
		}
		return s / float64(len(a)), nil
	}},
}

var constants = map[string]float64{
	"e": math.E, "pi": math.Pi, "π": math.Pi, "tau": 2 * math.Pi, "phi": math.Phi,
}

func factorial(x float64) (float64, error) {
	if x < 0 || x != math.Floor(x) {
		return 0, &Error{Msg: "factorial needs a non-negative integer"}
	}
	if x > 170 {
		return math.Inf(1), nil
	}
	r := 1.0
	for i := 2.0; i <= x; i++ {
		r *= i
	}
	return r, nil
}
