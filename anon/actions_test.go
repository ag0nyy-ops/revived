package main

import "testing"

func act(t *testing.T, r Rule, cfg *Config, in string) (string, error) {
	t.Helper()
	if cfg == nil {
		cfg = &Config{}
	}
	fn, err := newAction(&r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return fn(in)
}

func TestHashDeterministicAndSalted(t *testing.T) {
	a, _ := act(t, Rule{Action: "hash", Salt: "x"}, nil, "bob")
	b, _ := act(t, Rule{Action: "hash", Salt: "x"}, nil, "bob")
	c, _ := act(t, Rule{Action: "hash", Salt: "y"}, nil, "bob")
	if a != b || a == c || len(a) != 64 {
		t.Fatalf("bad hashes %s %s %s", a, b, c)
	}
}

func TestHashNeedsSalt(t *testing.T) {
	if _, err := newAction(&Rule{Action: "hash"}, &Config{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := newAction(&Rule{Action: "hash"}, &Config{Salt: "s"}); err != nil {
		t.Fatal(err)
	}
}

func TestOutcode(t *testing.T) {
	for in, want := range map[string]string{"W1W 8BE": "W1W", "sw1a1aa": "SW1A", "M1 1AE": "M1", "": ""} {
		got, err := act(t, Rule{Action: "outcode"}, nil, in)
		if err != nil || got != want {
			t.Errorf("%q: got %q %v", in, got, err)
		}
	}
	if _, err := act(t, Rule{Action: "outcode"}, nil, "nope"); err == nil {
		t.Error("expected error")
	}
}

func TestDates(t *testing.T) {
	got, err := act(t, Rule{Action: "year", Format: "DD/MM/YYYY"}, nil, "03/02/1990")
	if err != nil || got != "1990" {
		t.Fatal(got, err)
	}
	got, _ = act(t, Rule{Action: "date", Format: "YYYY-MM-DD", OutputFormat: "YYYY-MM"}, nil, "2001-09-17")
	if got != "2001-09" {
		t.Fatal(got)
	}
	if _, err := act(t, Rule{Action: "year", Format: "YYYY-MM-DD"}, nil, "junk"); err == nil {
		t.Fatal("expected error")
	}
}

func f(v float64) *float64 { return &v }

func TestRange(t *testing.T) {
	r := Rule{Action: "range", Ranges: []Range{{Gte: f(0), Lt: f(100), Output: "0-100"}, {Gte: f(100), Output: "100+"}}}
	for in, want := range map[string]string{"0": "0-100", "99.9": "0-100", "100": "100+", "5000": "100+"} {
		if got, err := act(t, r, nil, in); err != nil || got != want {
			t.Errorf("%s: %q %v", in, got, err)
		}
	}
	if _, err := act(t, r, nil, "-1"); err == nil {
		t.Error("expected out of range error")
	}
	if _, err := newAction(&Rule{Action: "range", Ranges: []Range{{Gt: f(1), Gte: f(1), Output: "x"}}}, &Config{}); err == nil {
		t.Error("expected conflicting bounds error")
	}
}

func TestMaskRedactIP(t *testing.T) {
	if got, _ := act(t, Rule{Action: "mask", Keep: 4}, nil, "4111111111111111"); got != "************1111" {
		t.Error(got)
	}
	if got, _ := act(t, Rule{Action: "mask", Keep: 4}, nil, "12"); got != "**" {
		t.Error(got)
	}
	if got, _ := act(t, Rule{Action: "redact"}, nil, "secret"); got != "REDACTED" {
		t.Error(got)
	}
	for in, want := range map[string]string{"192.168.4.77": "192.168.4.0", "2001:db8:1:2:3:4:5:6": "2001:db8:1::", "::ffff:10.1.2.3": "10.1.2.0"} {
		if got, err := act(t, Rule{Action: "ip"}, nil, in); err != nil || got != want {
			t.Errorf("%s: %q %v", in, got, err)
		}
	}
	if _, err := act(t, Rule{Action: "ip"}, nil, "x"); err == nil {
		t.Error("expected error")
	}
}

func TestUnknownAction(t *testing.T) {
	if _, err := newAction(&Rule{Action: "boom"}, &Config{}); err == nil {
		t.Fatal("expected error")
	}
}
