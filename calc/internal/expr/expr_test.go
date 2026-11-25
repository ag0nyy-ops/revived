package expr

import (
	"math"
	"strings"
	"testing"
)

func TestEval(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"1+1", 2},
		{"3(5/(3-4))", -15},
		{"3pi^2", 3 * math.Pi * math.Pi},
		{"-2^2", -4},
		{"2^-2", 0.25},
		{"2^3^2", 512},
		{"(1+2)(3+4)", 21},
		{"2 * 3 + 4", 10},
		{"10 % 4", 2},
		{"5!", 120},
		{"3!^2", 36},
		{"1_000 + 1", 1001},
		{"1e3 + .5", 1000.5},
		{"sqrt(16)", 4},
		{"max(1, 7, 3)", 7},
		{"avg(2,4)", 3},
		{"floor(-1.5)", -2},
		{"round(2.5)", 3},
		{"2π", 2 * math.Pi},
		{"6 ÷ 3 × 2", 4},
		{"--3", 3},
		{"ln(e)", 1},
		{"log(1000)", 3},
	}
	for _, c := range cases {
		got, err := NewEnv().Eval(c.in)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-12 {
			t.Errorf("%q = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestHistoryAndVars(t *testing.T) {
	e := NewEnv()
	steps := []struct {
		in   string
		want float64
	}{
		{"x = 4", 4},
		{"x * 2", 8},
		{"@ + 1", 9},
		{"@@ + @", 17},
		{"ans * 2", 34},
		{"y = x + 1", 5},
		{"x y", 20},
	}
	for _, s := range steps {
		got, err := e.Eval(s.in)
		if err != nil || got != s.want {
			t.Fatalf("%q = %v, %v; want %v", s.in, got, err, s.want)
		}
	}
}

func TestErrors(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "empty"},
		{"1+", "end of expression"},
		{"(2", "unclosed"},
		{"2)", "unmatched"},
		{"foo", "unknown name"},
		{"1/0", "division by zero"},
		{"5 % 0", "modulo by zero"},
		{"ln(-1)", "not a number"},
		{"sqrt()", "wrong number"},
		{"sin 1", "parentheses"},
		{"pi = 3", "cannot assign"},
		{"sin = 3", "cannot assign"},
		{"@", "no result"},
		{"1 $ 2", "unexpected character"},
		{"(-1)!", "factorial"},
		{"1.2.3", "invalid number"},
	}
	for _, c := range cases {
		_, err := NewEnv().Eval(c.in)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %v, want containing %q", c.in, err, c.want)
		}
	}
}

func TestErrorColumn(t *testing.T) {
	_, err := NewEnv().Eval("1 + $")
	e, ok := err.(*Error)
	if !ok || e.Pos != 5 {
		t.Fatalf("got %#v, want column 5", err)
	}
}

func TestFailedEvalKeepsHistory(t *testing.T) {
	e := NewEnv()
	e.Eval("2")
	e.Eval("1/0")
	if v, _ := e.Eval("@"); v != 2 {
		t.Fatalf("got %v", v)
	}
}

func TestFormat(t *testing.T) {
	cases := map[float64]string{
		2: "2", -15: "-15", 0.25: "0.25", 1e20: "1e+20",
		math.Inf(1): "inf", math.Inf(-1): "-inf",
	}
	for in, want := range cases {
		if got := Format(in); got != want {
			t.Errorf("Format(%v) = %q, want %q", in, got, want)
		}
	}
}
