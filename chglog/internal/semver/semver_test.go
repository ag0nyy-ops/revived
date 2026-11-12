package semver

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in, prefix, want string
		wantErr          bool
	}{
		{"1.2.3", "", "1.2.3", false},
		{"v1.2.3", "", "v1.2.3", false},
		{"v1.0.0-rc.1+build.5", "", "v1.0.0-rc.1+build.5", false},
		{"api/v2.0.0", "api/v", "api/v2.0.0", false},
		{"web/v2.0.0", "api/v", "", true},
		{"1.2", "", "", true},
		{"01.2.3", "", "", true},
		{"1.2.3-", "", "", true},
		{"1.2.3-a..b", "", "", true},
		{"1.2.3+", "", "", true},
		{"latest", "", "", true},
	}
	for _, tt := range tests {
		v, err := Parse(tt.in, tt.prefix)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q, %q) err = %v, wantErr %v", tt.in, tt.prefix, err, tt.wantErr)
			continue
		}
		if err == nil && v.String() != tt.want {
			t.Errorf("Parse(%q).String() = %q, want %q", tt.in, v.String(), tt.want)
		}
	}
}

func TestCompareSpecOrder(t *testing.T) {
	// Precedence example straight from semver.org section 11.
	order := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta",
		"1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "2.0.0",
	}
	for i := 0; i+1 < len(order); i++ {
		a, _ := Parse(order[i], "")
		b, _ := Parse(order[i+1], "")
		if Compare(a, b) != -1 || Compare(b, a) != 1 {
			t.Errorf("want %s < %s", order[i], order[i+1])
		}
	}
	a, _ := Parse("1.0.0+a", "")
	b, _ := Parse("v1.0.0+b", "")
	if Compare(a, b) != 0 {
		t.Error("build metadata and prefix must not affect precedence")
	}
}

func TestBump(t *testing.T) {
	tests := []struct {
		in    string
		level Level
		want  string
	}{
		{"v1.2.3", Patch, "v1.2.4"},
		{"v1.2.3", Minor, "v1.3.0"},
		{"v1.2.3", Major, "v2.0.0"},
		{"v1.2.3", None, "v1.2.3"},
		{"v0.4.2", Major, "v0.5.0"}, // breaking before 1.0 bumps minor
		{"v1.3.0-rc.1", Minor, "v1.3.0"},
		{"v1.3.0-rc.1", Patch, "v1.3.0"},
		{"v1.3.1-rc.1", Minor, "v1.4.0"},
		{"v2.0.0-beta.1", Major, "v2.0.0"},
		{"1.0.0+meta", Patch, "1.0.1"},
	}
	for _, tt := range tests {
		v, err := Parse(tt.in, "")
		if err != nil {
			t.Fatal(err)
		}
		if got := Bump(v, tt.level).String(); got != tt.want {
			t.Errorf("Bump(%s, %s) = %s, want %s", tt.in, tt.level, got, tt.want)
		}
	}
}
