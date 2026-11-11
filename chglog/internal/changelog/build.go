// Package changelog groups commits into releases and renders them.
package changelog

import (
	"regexp"
	"sort"

	"github.com/ag0nyy-ops/revived/chglog/internal/commit"
	"github.com/ag0nyy-ops/revived/chglog/internal/gitlog"
	"github.com/ag0nyy-ops/revived/chglog/internal/semver"
)

// Release is one section of the changelog. The unreleased section has an
// empty Tag unless a next tag was given.
type Release struct {
	Tag        string          `json:"tag"`
	PrevTag    string          `json:"previous_tag,omitempty"`
	Date       string          `json:"date,omitempty"`
	Unreleased bool            `json:"unreleased"`
	Commits    []commit.Commit `json:"commits"`
}

// Options controls how tags are selected.
type Options struct {
	TagPrefix        string         // only tags starting with this are versions; "" accepts an optional "v"
	TagFilter        *regexp.Regexp // optional extra filter on tag names
	SkipPrereleases  bool           // fold prerelease tags into the next real release
	OnlyCommits      map[string]bool
	IncludeMerges    bool
	KeepRevertedPair bool // keep a commit and its revert when both are in one release
}

// VersionTag is a tag that parsed as a semantic version.
type VersionTag struct {
	gitlog.Tag
	Version semver.Version
}

// VersionTags keeps the tags that parse as versions under opts, sorted oldest
// version first. When two tags carry the same version, the first by name wins.
func VersionTags(tags []gitlog.Tag, opts Options) []VersionTag {
	var out []VersionTag
	for _, t := range tags {
		if opts.TagFilter != nil && !opts.TagFilter.MatchString(t.Name) {
			continue
		}
		v, err := semver.Parse(t.Name, opts.TagPrefix)
		if err != nil {
			continue
		}
		if opts.SkipPrereleases && len(v.Prerelease) > 0 {
			continue
		}
		out = append(out, VersionTag{Tag: t, Version: v})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if c := semver.Compare(out[i].Version, out[j].Version); c != 0 {
			return c < 0
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Build assigns every commit to the oldest release that contains it, walking
// the commit graph in memory, and returns releases newest first. Commits
// reachable from head but from no tag form the unreleased section, which is
// returned even when empty so callers can label it with a next version.
func Build(commits []commit.Commit, tags []VersionTag, head string, opts Options) []Release {
	byHash := make(map[string]*commit.Commit, len(commits))
	order := make(map[string]int, len(commits))
	for i := range commits {
		byHash[commits[i].Hash] = &commits[i]
		order[commits[i].Hash] = i
	}
	owner := map[string]int{} // commit hash -> index into releases
	var releases []Release

	claim := func(start string, idx int) {
		stack := []string{start}
		for len(stack) > 0 {
			h := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, done := owner[h]; done {
				continue
			}
			c, ok := byHash[h]
			if !ok {
				continue
			}
			owner[h] = idx
			stack = append(stack, c.Parents...)
		}
	}

	prev := ""
	for _, t := range tags {
		if _, ok := byHash[t.Commit]; !ok {
			continue
		}
		releases = append(releases, Release{Tag: t.Name, PrevTag: prev, Date: day(t.Date)})
		claim(t.Commit, len(releases)-1)
		prev = t.Name
	}
	releases = append(releases, Release{PrevTag: prev, Unreleased: true})
	if head != "" {
		claim(head, len(releases)-1)
	}

	hashes := make([]string, 0, len(owner))
	for h := range owner {
		hashes = append(hashes, h)
	}
	sort.Slice(hashes, func(i, j int) bool { return order[hashes[i]] < order[hashes[j]] })
	for _, h := range hashes {
		c := byHash[h]
		if c.IsMerge() && !opts.IncludeMerges {
			continue
		}
		if opts.OnlyCommits != nil && !opts.OnlyCommits[h] {
			continue
		}
		r := &releases[owner[h]]
		r.Commits = append(r.Commits, *c)
	}
	if !opts.KeepRevertedPair {
		for i := range releases {
			releases[i].Commits = dropRevertedPairs(releases[i].Commits)
		}
	}

	// newest first
	for i, j := 0, len(releases)-1; i < j; i, j = i+1, j-1 {
		releases[i], releases[j] = releases[j], releases[i]
	}
	return releases
}

// dropRevertedPairs removes commits that were reverted inside the same
// release together with the revert itself: neither is a user-visible change.
func dropRevertedPairs(cs []commit.Commit) []commit.Commit {
	drop := map[string]bool{}
	for _, c := range cs {
		if !c.Revert {
			continue
		}
		for _, o := range cs {
			if o.Hash == c.Hash || drop[o.Hash] {
				continue
			}
			if (c.RevertsHash != "" && hasPrefix(o.Hash, c.RevertsHash)) ||
				(c.RevertsHash == "" && o.Header == c.Subject) {
				drop[o.Hash], drop[c.Hash] = true, true
				break
			}
		}
	}
	if len(drop) == 0 {
		return cs
	}
	out := cs[:0:0]
	for _, c := range cs {
		if !drop[c.Hash] {
			out = append(out, c)
		}
	}
	return out
}

func hasPrefix(full, short string) bool {
	return len(short) <= len(full) && full[:len(short)] == short
}

func day(iso string) string {
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}

// BumpLevel is the release level implied by a set of commits:
// breaking -> major, feat -> minor, fix/perf/revert -> patch.
func BumpLevel(cs []commit.Commit) semver.Level {
	l := semver.None
	for _, c := range cs {
		switch {
		case c.Breaking:
			return semver.Major
		case c.Type == "feat":
			l = max(l, semver.Minor)
		case c.Type == "fix" || c.Type == "perf" || c.Type == "revert":
			l = max(l, semver.Patch)
		}
	}
	return l
}
