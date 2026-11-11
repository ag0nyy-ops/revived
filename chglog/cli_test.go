package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repo builds a throwaway git repo and returns a helper that runs git in it.
func repo(t *testing.T) (string, func(args ...string) string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for k, v := range map[string]string{
		"GIT_AUTHOR_NAME": "Test", "GIT_AUTHOR_EMAIL": "t@example.com",
		"GIT_COMMITTER_NAME": "Test", "GIT_COMMITTER_EMAIL": "t@example.com",
		"GIT_AUTHOR_DATE": "2026-01-02T03:04:05Z", "GIT_COMMITTER_DATE": "2026-01-02T03:04:05Z",
	} {
		t.Setenv(k, v)
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("config", "commit.gpgsign", "false")
	git("config", "tag.gpgsign", "false")
	return dir, git
}

func commitFile(t *testing.T, dir string, git func(...string) string, file, msg string) {
	t.Helper()
	p := filepath.Join(dir, file)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(msg + "\n")
	f.Close()
	git("add", "-A")
	git("commit", "-q", "-m", msg)
}

func chglog(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	err := run(args, &out, &errb)
	return out.String(), errb.String(), err
}

func TestEndToEnd(t *testing.T) {
	dir, git := repo(t)
	git("remote", "add", "origin", "git@github.com:acme/widget.git")
	commitFile(t, dir, git, "a.txt", "chore: init")
	commitFile(t, dir, git, "api/x.go", "feat(api): add widgets endpoint")
	git("tag", "v0.1.0")

	// a feature branch written before v0.2.0 but merged after it
	git("checkout", "-q", "-b", "feature")
	commitFile(t, dir, git, "api/y.go", "feat(api): search widgets")
	git("checkout", "-q", "main")

	commitFile(t, dir, git, "a.txt", "fix: handle empty body (#3)")
	commitFile(t, dir, git, "a.txt", "feat!: new config format\n\nBREAKING CHANGE: config.yml is now config.toml")
	git("tag", "-a", "v0.2.0", "-m", "release 0.2.0")

	git("merge", "-q", "--no-ff", "feature", "-m", "Merge branch 'feature'")
	commitFile(t, dir, git, "a.txt", "feat: experimental thing")
	git("revert", "--no-edit", "HEAD")
	commitFile(t, dir, git, "a.txt", "perf: faster lookups")

	out, _, err := chglog(t, "-C", dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# Changelog",
		"## [Unreleased](https://github.com/acme/widget/compare/v0.2.0...HEAD)",
		"- **api:** search widgets ([",
		"### Performance\n\n- faster lookups",
		"## [v0.2.0](https://github.com/acme/widget/compare/v0.1.0...v0.2.0) (2026-01-02)",
		"### ⚠ BREAKING CHANGES\n\n- config.yml is now config.toml",
		"- handle empty body ([#3](https://github.com/acme/widget/issues/3))",
		"## v0.1.0 (2026-01-02)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"experimental thing", "Merge branch", "chore: init", "init ("} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, out)
		}
	}
	// search widgets must be unreleased, not part of v0.2.0
	if strings.Index(out, "search widgets") > strings.Index(out, "## [v0.2.0]") {
		t.Errorf("branch commit merged after v0.2.0 landed in the wrong release:\n%s", out)
	}

	bump, _, err := chglog(t, "-C", dir, "--bump")
	if err != nil || strings.TrimSpace(bump) != "v0.3.0" {
		t.Errorf("--bump = %q, %v", bump, err)
	}

	notes, _, err := chglog(t, "-C", dir, "--next-tag", "auto", "--unreleased", "--no-links")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(notes, "## v0.3.0 (") || strings.Contains(notes, "# Changelog") || strings.Contains(notes, "v0.2.0") {
		t.Errorf("unreleased notes:\n%s", notes)
	}

	latest, _, err := chglog(t, "-C", dir, "--latest", "--no-links")
	if err != nil || !strings.HasPrefix(latest, "## v0.2.0 (2026-01-02)") || strings.Contains(latest, "v0.1.0") {
		t.Errorf("--latest = %q, %v", latest, err)
	}

	api, _, err := chglog(t, "-C", dir, "--path", "api", "--no-links", "--title", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(api, "faster lookups") || !strings.Contains(api, "search widgets") || !strings.Contains(api, "add widgets endpoint") {
		t.Errorf("--path api:\n%s", api)
	}

	js, _, err := chglog(t, "-C", dir, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var rel []struct {
		Tag        string `json:"tag"`
		Unreleased bool   `json:"unreleased"`
		Commits    []struct {
			Type    string `json:"type"`
			Subject string `json:"subject"`
		} `json:"commits"`
	}
	if err := json.Unmarshal([]byte(js), &rel); err != nil {
		t.Fatalf("bad json: %v\n%s", err, js)
	}
	if len(rel) != 3 || !rel[0].Unreleased || rel[1].Tag != "v0.2.0" || rel[2].Tag != "v0.1.0" {
		t.Errorf("json releases: %+v", rel)
	}

	file := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if _, _, err := chglog(t, "-C", dir, "-o", file); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(file); string(b) != out {
		t.Error("-o output differs from stdout output")
	}
}

func TestNoTagsAndAllTypes(t *testing.T) {
	dir, git := repo(t)
	commitFile(t, dir, git, "a", "docs: write readme")
	commitFile(t, dir, git, "a", "Initial import")

	out, _, err := chglog(t, "-C", dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Unreleased") {
		t.Errorf("nothing notable, but got:\n%s", out)
	}
	out, _, err = chglog(t, "-C", dir, "--all")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "### Documentation") || !strings.Contains(out, "### Other\n\n- Initial import") {
		t.Errorf("--all:\n%s", out)
	}
	bump, stderr, _ := chglog(t, "-C", dir, "--bump")
	if strings.TrimSpace(bump) != "v0.0.0" || !strings.Contains(stderr, "no release-worthy") {
		t.Errorf("--bump with nothing to release = %q / %q", bump, stderr)
	}
}

func TestErrors(t *testing.T) {
	dir, _ := repo(t)
	if _, _, err := chglog(t, "-C", dir); err == nil || !strings.Contains(err.Error(), "no commits") {
		t.Errorf("empty repo: %v", err)
	}
	if _, _, err := chglog(t, "-C", t.TempDir()); err == nil {
		t.Error("expected error outside a git repo")
	}
	if _, _, err := chglog(t, "--format", "xml"); err == nil {
		t.Error("expected error for bad format")
	}
	if _, _, err := chglog(t, "--latest", "--unreleased"); err == nil {
		t.Error("expected error for conflicting flags")
	}
	if _, _, err := chglog(t, "extra"); err == nil {
		t.Error("expected error for stray argument")
	}
	if out, _, err := chglog(t, "--version"); err != nil || !strings.HasPrefix(out, "chglog ") {
		t.Errorf("--version = %q, %v", out, err)
	}
}
