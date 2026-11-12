package changelog

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ag0nyy-ops/revived/chglog/internal/commit"
	"github.com/ag0nyy-ops/revived/chglog/internal/gitlog"
	"github.com/ag0nyy-ops/revived/chglog/internal/semver"
)

func mk(hash, msg string, parents ...string) commit.Commit {
	return commit.Parse(commit.Commit{Hash: hash, Parents: parents}, msg)
}

func hashes(r Release) string {
	var hs []string
	for _, c := range r.Commits {
		hs = append(hs, c.Hash)
	}
	return strings.Join(hs, ",")
}

// History (newest first):
//
//	m  merge of b into main (after v1.1.0)
//	f  fix on main            <- v1.1.0
//	b  feat on a branch, started from a
//	a  feat                    <- v1.0.0
//
// b was written before v1.1.0 but only merged after it, so it belongs to the
// unreleased section, not v1.1.0. Date-based tools get this wrong.
func graph() []commit.Commit {
	return []commit.Commit{
		mk("m", "Merge branch 'x'", "f", "b"),
		mk("f", "fix: patch", "a"),
		mk("b", "feat: branch work", "a"),
		mk("a", "feat: first"),
	}
}

func TestBuildAssignsByReachability(t *testing.T) {
	tags := VersionTags([]gitlog.Tag{
		{Name: "v1.1.0", Commit: "f", Date: "2026-02-01T00:00:00Z"},
		{Name: "v1.0.0", Commit: "a", Date: "2026-01-01T00:00:00Z"},
		{Name: "nightly", Commit: "b"},
	}, Options{})
	rs := Build(graph(), tags, "m", Options{})
	if len(rs) != 3 {
		t.Fatalf("got %d releases", len(rs))
	}
	if !rs[0].Unreleased || hashes(rs[0]) != "b" {
		t.Errorf("unreleased = %q, want b (merge commit skipped)", hashes(rs[0]))
	}
	if rs[1].Tag != "v1.1.0" || rs[1].PrevTag != "v1.0.0" || hashes(rs[1]) != "f" || rs[1].Date != "2026-02-01" {
		t.Errorf("v1.1.0 = %+v", rs[1])
	}
	if rs[2].Tag != "v1.0.0" || rs[2].PrevTag != "" || hashes(rs[2]) != "a" {
		t.Errorf("v1.0.0 = %+v", rs[2])
	}
}

func TestBuildIncludeMergesAndPathFilter(t *testing.T) {
	rs := Build(graph(), nil, "m", Options{IncludeMerges: true, OnlyCommits: map[string]bool{"m": true, "a": true}})
	if len(rs) != 1 || hashes(rs[0]) != "m,a" {
		t.Errorf("got %q", hashes(rs[0]))
	}
}

func TestVersionTagsSortingAndFilters(t *testing.T) {
	raw := []gitlog.Tag{{Name: "v1.10.0"}, {Name: "v1.2.0"}, {Name: "v1.2.0-rc.1"}, {Name: "web/v9.0.0"}, {Name: "junk"}}
	names := func(ts []VersionTag) string {
		var s []string
		for _, t := range ts {
			s = append(s, t.Name)
		}
		return strings.Join(s, " ")
	}
	if got := names(VersionTags(raw, Options{})); got != "v1.2.0-rc.1 v1.2.0 v1.10.0" {
		t.Errorf("default: %s", got)
	}
	if got := names(VersionTags(raw, Options{SkipPrereleases: true})); got != "v1.2.0 v1.10.0" {
		t.Errorf("skip prereleases: %s", got)
	}
	if got := names(VersionTags(raw, Options{TagPrefix: "web/v"})); got != "web/v9.0.0" {
		t.Errorf("prefix: %s", got)
	}
	if got := names(VersionTags(raw, Options{TagFilter: regexp.MustCompile(`^v1\.2`)})); got != "v1.2.0-rc.1 v1.2.0" {
		t.Errorf("filter: %s", got)
	}
}

func TestRevertedPairsAreDropped(t *testing.T) {
	cs := []commit.Commit{
		mk("r2", "Revert \"feat: keep me\"", "r1"),
		mk("r1", "Revert \"feat: oops\"\n\nThis reverts commit c2c2c2c.", "c2c2c2cfff"),
		mk("c2c2c2cfff", "feat: oops", "c1"),
		mk("c1", "feat: keep me"),
	}
	rs := Build(cs, nil, "r2", Options{})
	if got := hashes(rs[0]); got != "" {
		t.Errorf("got %q, want both pairs dropped", got)
	}
	rs = Build(cs, nil, "r2", Options{KeepRevertedPair: true})
	if got := hashes(rs[0]); got != "r2,r1,c2c2c2cfff,c1" {
		t.Errorf("keep: got %q", got)
	}
}

func TestBumpLevel(t *testing.T) {
	tests := []struct {
		msgs []string
		want semver.Level
	}{
		{[]string{"docs: x", "chore: y"}, semver.None},
		{[]string{"docs: x", "fix: y"}, semver.Patch},
		{[]string{"perf: x"}, semver.Patch},
		{[]string{"fix: y", "feat: z"}, semver.Minor},
		{[]string{"feat: z", "chore!: drop node 18"}, semver.Major},
		{[]string{"fix: y\n\nBREAKING CHANGE: different"}, semver.Major},
	}
	for _, tt := range tests {
		var cs []commit.Commit
		for _, m := range tt.msgs {
			cs = append(cs, mk("h", m))
		}
		if got := BumpLevel(cs); got != tt.want {
			t.Errorf("%v: got %s, want %s", tt.msgs, got, tt.want)
		}
	}
}

func TestMarkdown(t *testing.T) {
	rs := []Release{
		{Tag: "v2.0.0", PrevTag: "v1.0.0", Date: "2026-03-04", Commits: []commit.Commit{
			mk("1111111aaaa", "feat(api)!: new ids\n\nBREAKING CHANGE: ids are strings\nnow"),
			mk("2222222bbbb", "fix: crash (#1)\n\nCloses #12"),
			mk("3333333cccc", "docs: typo"),
			mk("4444444dddd", "Bump version"),
		}},
		{Tag: "v1.0.0", Date: "2026-01-01", Commits: []commit.Commit{mk("5555555eeee", "chore: init")}},
	}
	links := gitlog.Links{Base: "https://github.com/a/b", Style: "github"}
	var b strings.Builder
	if err := Markdown(&b, rs, MarkdownOptions{Title: "Changelog", Sections: DefaultSections, Links: links}); err != nil {
		t.Fatal(err)
	}
	want := `# Changelog

## [v2.0.0](https://github.com/a/b/compare/v1.0.0...v2.0.0) (2026-03-04)

### ⚠ BREAKING CHANGES

- **api:** ids are strings
  now

### Features

- **api:** new ids ([1111111](https://github.com/a/b/commit/1111111aaaa))

### Bug Fixes

- crash ([#1](https://github.com/a/b/issues/1)) ([2222222](https://github.com/a/b/commit/2222222bbbb)), closes [#12](https://github.com/a/b/issues/12)

## v1.0.0 (2026-01-01)

_No notable changes._
`
	if b.String() != want {
		t.Errorf("markdown mismatch\n--- got\n%s\n--- want\n%s", b.String(), want)
	}

	b.Reset()
	if err := Markdown(&b, rs[:1], MarkdownOptions{Sections: AllSections}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"### Documentation\n\n- typo (3333333)", "### Other\n\n- Bump version (4444444)", "- crash (#1) (2222222), closes #12"} {
		if !strings.Contains(b.String(), s) {
			t.Errorf("all sections output missing %q:\n%s", s, b.String())
		}
	}
	if strings.HasPrefix(b.String(), "#  ") || strings.HasPrefix(b.String(), "# Changelog") {
		t.Error("empty title should be omitted")
	}
}

func TestEmptyUnreleasedIsHidden(t *testing.T) {
	var b strings.Builder
	_ = Markdown(&b, []Release{{Unreleased: true}, {Tag: "v1.0.0", Commits: []commit.Commit{mk("a", "feat: x")}}}, MarkdownOptions{Sections: DefaultSections})
	if strings.Contains(b.String(), "Unreleased") {
		t.Errorf("empty unreleased section rendered:\n%s", b.String())
	}
}
