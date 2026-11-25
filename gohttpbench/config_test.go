package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgsDefaults(t *testing.T) {
	c, err := parseArgs([]string{"http://localhost:8080/x"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if c.Method != "GET" || c.Requests != 1 || c.Concurrency != 1 {
		t.Fatalf("%+v", c)
	}
}

func TestParseArgsBodyFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "body")
	os.WriteFile(f, []byte("hello"), 0o600)
	c, err := parseArgs([]string{"-n", "2", "-p", f, "-T", "application/json", "http://h/"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if c.Method != "POST" || string(c.Body) != "hello" || c.ContentType != "application/json" {
		t.Fatalf("%+v", c)
	}
	c, _ = parseArgs([]string{"-u", f, "http://h/"}, io.Discard)
	if c.Method != "PUT" {
		t.Fatalf("method %s", c.Method)
	}
}

func TestParseArgsRepeatableFlags(t *testing.T) {
	c, err := parseArgs([]string{"-H", "A: 1", "-H", "B: 2", "-C", "x=1", "-i", "http://h/"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Headers) != 2 || len(c.Cookies) != 1 || c.Method != "HEAD" {
		t.Fatalf("%+v", c)
	}
}

func TestParseArgsErrors(t *testing.T) {
	cases := map[string][]string{
		"no url":            {},
		"bad scheme":        {"ftp://h/"},
		"c over n":          {"-n", "2", "-c", "5", "http://h/"},
		"zero c":            {"-c", "0", "http://h/"},
		"bad header":        {"-H", "nocolon", "http://h/"},
		"bad cookie":        {"-C", "nope", "http://h/"},
		"bad auth":          {"-A", "nopass", "http://h/"},
		"missing file":      {"-p", "/does/not/exist", "http://h/"},
		"head with post":    {"-i", "-p", "/dev/null", "http://h/"},
		"negative duration": {"-d", "-1s", "http://h/"},
	}
	for name, args := range cases {
		var sb strings.Builder
		if _, err := parseArgs(args, &sb); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestDurationModeAllowsConcurrencyAboveRequests(t *testing.T) {
	if _, err := parseArgs([]string{"-d", "1s", "-c", "20", "http://h/"}, io.Discard); err != nil {
		t.Fatal(err)
	}
}
