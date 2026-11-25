package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(args []string, stdin string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestCommandMode(t *testing.T) {
	code, out, _ := runCLI([]string{"2", "+", "3"}, "")
	if code != 0 || out != "5\n" {
		t.Fatalf("got %d %q", code, out)
	}
}

func TestCommandError(t *testing.T) {
	code, out, errs := runCLI([]string{"1/0"}, "")
	if code != 1 || out != "" || !strings.Contains(errs, "division by zero") {
		t.Fatalf("got %d %q %q", code, out, errs)
	}
}

func TestPipedStdin(t *testing.T) {
	code, out, errs := runCLI(nil, "# comment\nx=2\n\nx^10\n1+\n@\nexit\n99\n")
	if code != 1 {
		t.Fatalf("exit %d, want 1 after a bad line", code)
	}
	if out != "2\n1024\n1024\n" {
		t.Fatalf("out %q", out)
	}
	if !strings.Contains(errs, "error: column") {
		t.Fatalf("err %q", errs)
	}
}

func TestHelp(t *testing.T) {
	code, out, _ := runCLI([]string{"--help"}, "")
	if code != 0 || !strings.Contains(out, "usage: calc") {
		t.Fatalf("got %d %q", code, out)
	}
}
