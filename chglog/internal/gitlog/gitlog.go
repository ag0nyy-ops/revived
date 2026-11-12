// Package gitlog reads commits and tags from a git repository by shelling out
// to the git binary. The whole history is read with a single `git log` call.
package gitlog

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ag0nyy-ops/revived/chglog/internal/commit"
)

// Repo is a git working tree or bare repository on disk.
type Repo struct {
	Dir string
	Git string // git binary, defaults to "git"
}

// Tag is a tag name and the commit it points to (annotated tags are peeled).
type Tag struct {
	Name   string
	Commit string
	Date   string // ISO 8601: tagger date for annotated tags, commit date otherwise
}

func (r Repo) run(args ...string) (string, error) {
	bin := r.Git
	if bin == "" {
		bin = "git"
	}
	cmd := exec.Command(bin, append([]string{"-C", r.Dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

// ErrNoCommits is returned when HEAD does not point at a commit yet.
var ErrNoCommits = errors.New("repository has no commits yet")

// Head returns the commit hash HEAD points to.
func (r Repo) Head() (string, error) {
	out, err := r.run("rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err != nil {
		if _, err2 := r.run("rev-parse", "--git-dir"); err2 != nil {
			return "", err2
		}
		return "", ErrNoCommits
	}
	return strings.TrimSpace(out), nil
}

// Tags lists every tag with the commit it ultimately points to. Tags that
// point at trees or blobs are skipped.
func (r Repo) Tags() ([]Tag, error) {
	out, err := r.run("for-each-ref",
		"--format=%(refname:strip=2)%1f%(objecttype)%1f%(objectname)%1f%(*objecttype)%1f%(*objectname)%1f%(creatordate:iso-strict)",
		"refs/tags")
	if err != nil {
		return nil, err
	}
	var tags []Tag
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\x1f")
		if len(f) != 6 {
			return nil, fmt.Errorf("unexpected for-each-ref output %q", line)
		}
		t := Tag{Name: f[0], Date: f[5]}
		switch {
		case f[1] == "commit":
			t.Commit = f[2]
		case f[1] == "tag" && f[3] == "commit":
			t.Commit = f[4]
		default:
			continue
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// Commits returns every commit reachable from HEAD or any tag, newest first,
// with parents filled in so callers can walk the graph.
func (r Repo) Commits() ([]commit.Commit, error) {
	out, err := r.run("log", "HEAD", "--tags", "--topo-order",
		"--format=%H%x1f%P%x1f%an%x1f%aI%x1f%B%x1e")
	if err != nil {
		return nil, err
	}
	var commits []commit.Commit
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if rec == "" {
			continue
		}
		f := strings.SplitN(rec, "\x1f", 5)
		if len(f) != 5 {
			return nil, fmt.Errorf("unexpected git log record %q", rec)
		}
		c := commit.Commit{Hash: f[0], Author: f[2], Date: f[3]}
		if f[1] != "" {
			c.Parents = strings.Fields(f[1])
		}
		commits = append(commits, commit.Parse(c, f[4]))
	}
	return commits, nil
}

// Touching returns the set of commits that changed any of paths.
func (r Repo) Touching(paths []string) (map[string]bool, error) {
	args := append([]string{"log", "HEAD", "--tags", "--format=%H", "--"}, paths...)
	out, err := r.run(args...)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, h := range strings.Fields(out) {
		set[h] = true
	}
	return set, nil
}

// RemoteURL returns the fetch URL of the named remote, or "" if it isn't set.
func (r Repo) RemoteURL(name string) string {
	out, err := r.run("config", "--get", "remote."+name+".url")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
