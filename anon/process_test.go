package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cfgFrom(t *testing.T, js string) *Config {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(p, []byte(js), 0o600)
	c, err := LoadConfig(p, "")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func runCfg(c *Config, in string) (string, error) {
	var b bytes.Buffer
	err := Process(c, strings.NewReader(in), &b)
	return b.String(), err
}

func TestHeaderAndPassThrough(t *testing.T) {
	c := cfgFrom(t, `{"rules":[{"column":"zip","action":"outcode"},{"column":"card","action":"mask","keep":2}]}`)
	got, err := runCfg(c, "name,zip,card\nann,W1W 8BE,1234\n")
	if err != nil || got != "name,zip,card\nann,W1W,**34\n" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestNoHeaderIndexes(t *testing.T) {
	c := cfgFrom(t, `{"header":false,"delimiter":";","rules":[{"column":1,"action":"redact"}]}`)
	got, err := runCfg(c, "a;b;c\nd;e;f\n")
	if err != nil || got != "a;REDACTED;c\nd;REDACTED;f\n" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestNamesNeedHeader(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(p, []byte(`{"header":false,"rules":[{"column":"x","action":"redact"}]}`), 0o600)
	if _, err := LoadConfig(p, ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvSalt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(p, []byte(`{"rules":[{"column":0,"action":"hash"}]}`), 0o600)
	if _, err := LoadConfig(p, ""); err == nil {
		t.Fatal("expected missing salt error")
	}
	if _, err := LoadConfig(p, "s3"); err != nil {
		t.Fatal(err)
	}
}

func TestErrorsHaveLineAndColumn(t *testing.T) {
	c := cfgFrom(t, `{"rules":[{"column":"d","action":"year","format":"YYYY-MM-DD"}]}`)
	_, err := runCfg(c, "d\n2020-01-01\nbad\n")
	if err == nil || !strings.Contains(err.Error(), "line 3") || !strings.Contains(err.Error(), "column d") {
		t.Fatal(err)
	}
}

func TestMissingColumn(t *testing.T) {
	c := cfgFrom(t, `{"rules":[{"column":"nope","action":"redact"}]}`)
	if _, err := runCfg(c, "a\n1\n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestShortRow(t *testing.T) {
	c := cfgFrom(t, `{"rules":[{"column":"b","action":"redact"}]}`)
	if _, err := runCfg(c, "a,b\n1\n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSampling(t *testing.T) {
	c := cfgFrom(t, `{"sampling":{"mod":2,"column":"id"},"rules":[]}`)
	var in strings.Builder
	in.WriteString("id\n")
	for i := 0; i < 200; i++ {
		in.WriteString(strings.Repeat("x", i%7) + string(rune('a'+i%26)) + "\n")
	}
	got, err := runCfg(c, in.String())
	if err != nil {
		t.Fatal(err)
	}
	n := strings.Count(got, "\n")
	if n < 2 || n > 200 {
		t.Fatalf("sampled %d lines", n)
	}
	again, _ := runCfg(c, in.String())
	if again != got {
		t.Fatal("sampling must be deterministic")
	}
}

func TestUnknownConfigField(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(p, []byte(`{"rule":[]}`), 0o600)
	if _, err := LoadConfig(p, ""); err == nil {
		t.Fatal("expected error on typo")
	}
}

func TestCLI(t *testing.T) {
	dir := t.TempDir()
	cp := filepath.Join(dir, "c.json")
	os.WriteFile(cp, []byte(`{"rules":[{"column":"n","action":"hash","salt":"k"}]}`), 0o600)
	var out, errb bytes.Buffer
	code := run([]string{"-c", cp}, strings.NewReader("n,m\nbob,1\n"), &out, &errb)
	if code != 0 || !strings.HasPrefix(out.String(), "n,m\n") || strings.Contains(out.String(), "bob") {
		t.Fatal(code, out.String(), errb.String())
	}
	if run([]string{"-c", filepath.Join(dir, "none")}, strings.NewReader(""), &out, &errb) != 2 {
		t.Fatal("expected exit 2")
	}
}
