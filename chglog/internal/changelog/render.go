package changelog

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ag0nyy-ops/revived/chglog/internal/commit"
	"github.com/ag0nyy-ops/revived/chglog/internal/gitlog"
)

// Section maps a commit type to a heading.
type Section struct {
	Type  string
	Title string
}

// DefaultSections are shown unless all types are requested.
var DefaultSections = []Section{
	{"feat", "Features"},
	{"fix", "Bug Fixes"},
	{"perf", "Performance"},
	{"revert", "Reverts"},
}

// AllSections adds the housekeeping types and an "Other" bucket for
// commits that don't follow the convention.
var AllSections = append(append([]Section{}, DefaultSections...),
	Section{"refactor", "Refactoring"},
	Section{"docs", "Documentation"},
	Section{"style", "Style"},
	Section{"test", "Tests"},
	Section{"build", "Build"},
	Section{"ci", "CI"},
	Section{"chore", "Chores"},
	Section{"", "Other"},
)

// MarkdownOptions controls Markdown output.
type MarkdownOptions struct {
	Title    string // document heading; empty omits it
	Sections []Section
	Links    gitlog.Links
}

// Markdown writes releases as a Keep-a-Changelog-flavoured Markdown document.
func Markdown(w io.Writer, releases []Release, o MarkdownOptions) error {
	var b strings.Builder
	if o.Title != "" {
		fmt.Fprintf(&b, "# %s\n\n", o.Title)
	}
	known := map[string]bool{}
	for _, s := range o.Sections {
		known[s.Type] = true
	}
	for _, r := range releases {
		if r.Unreleased && r.Tag == "" && len(r.Commits) == 0 {
			continue
		}
		writeHeading(&b, r, o.Links)

		var breaking []commit.Commit
		for _, c := range r.Commits {
			if c.Breaking {
				breaking = append(breaking, c)
			}
		}
		wrote := false
		if len(breaking) > 0 {
			b.WriteString("### ⚠ BREAKING CHANGES\n\n")
			for _, c := range breaking {
				fmt.Fprintf(&b, "- %s%s\n", scopePrefix(c), indentNote(c.BreakingNote))
			}
			b.WriteString("\n")
			wrote = true
		}
		for _, s := range o.Sections {
			var lines []string
			for _, c := range r.Commits {
				t := c.Type
				if s.Type == "" { // "Other" catches anything without a listed section
					if known[t] && t != "" {
						continue
					}
				} else if t != s.Type {
					continue
				}
				lines = append(lines, commitLine(c, o.Links))
			}
			if len(lines) == 0 {
				continue
			}
			fmt.Fprintf(&b, "### %s\n\n%s\n\n", s.Title, strings.Join(lines, "\n"))
			wrote = true
		}
		if !wrote {
			b.WriteString("_No notable changes._\n\n")
		}
	}
	out := strings.TrimRight(b.String(), "\n") + "\n"
	_, err := io.WriteString(w, out)
	return err
}

func writeHeading(b *strings.Builder, r Release, l gitlog.Links) {
	name, to := r.Tag, r.Tag
	if name == "" {
		name, to = "Unreleased", "HEAD"
	}
	title := name
	if link := l.Compare(r.PrevTag, to); link != "" {
		title = fmt.Sprintf("[%s](%s)", name, link)
	}
	if r.Date != "" {
		fmt.Fprintf(b, "## %s (%s)\n\n", title, r.Date)
	} else {
		fmt.Fprintf(b, "## %s\n\n", title)
	}
}

func scopePrefix(c commit.Commit) string {
	if c.Scope == "" {
		return ""
	}
	return "**" + c.Scope + ":** "
}

func indentNote(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", "\n  ")
}

func commitLine(c commit.Commit, l gitlog.Links) string {
	short := c.Hash
	if len(short) > 7 {
		short = short[:7]
	}
	ref := short
	if u := l.Commit(c.Hash); u != "" {
		ref = fmt.Sprintf("[%s](%s)", short, u)
	}
	line := fmt.Sprintf("- %s%s (%s)", scopePrefix(c), c.Subject, ref)
	if len(c.Refs) > 0 {
		var refs []string
		for _, n := range c.Refs {
			if u := l.Issue(n); u != "" {
				refs = append(refs, fmt.Sprintf("[#%s](%s)", n, u))
			} else {
				refs = append(refs, "#"+n)
			}
		}
		// Don't repeat refs already visible in the subject, e.g. "(#12)".
		var extra []string
		for i, n := range c.Refs {
			if !strings.Contains(c.Subject, "#"+n) {
				extra = append(extra, refs[i])
			}
		}
		if len(extra) > 0 {
			line += ", closes " + strings.Join(extra, ", ")
		}
	}
	return line
}

// JSON writes releases as indented JSON.
func JSON(w io.Writer, releases []Release) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(releases)
}
