// Package commit parses commit messages following Conventional Commits 1.0.0.
package commit

import (
	"regexp"
	"strings"
)

// Commit is one parsed commit.
type Commit struct {
	Hash    string
	Author  string
	Date    string // ISO 8601, as reported by git
	Parents []string

	Type     string // lower-cased; empty when the header is not conventional
	Scope    string
	Subject  string
	Body     string
	Breaking bool
	// BreakingNote is the text of a BREAKING CHANGE footer, or the subject when
	// only "!" marked the commit as breaking.
	BreakingNote string
	Footers      []Footer
	Refs         []string // issue / PR numbers such as "12" from "Closes #12" or "(#12)"
	Revert       bool
}

// Footer is a "Token: value" or "Token #value" trailer.
type Footer struct {
	Token string
	Value string
}

// IsMerge reports whether the commit has more than one parent.
func (c Commit) IsMerge() bool { return len(c.Parents) > 1 }

// Conventional reports whether the header matched the spec.
func (c Commit) Conventional() bool { return c.Type != "" }

var (
	headerRe = regexp.MustCompile(`^(?P<type>[A-Za-z][\w-]*)(?:\((?P<scope>[^()\r\n]*)\))?(?P<bang>!)?: (?P<subject>.+)$`)
	footerRe = regexp.MustCompile(`^(BREAKING CHANGE|BREAKING-CHANGE|[A-Za-z][\w-]*)(?:: | #)(.*)$`)
	refRe    = regexp.MustCompile(`#(\d+)\b`)
	revertRe = regexp.MustCompile(`^Revert "(.+)"$`)
)

// Parse fills the conventional fields of c from a raw commit message.
func Parse(c Commit, message string) Commit {
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.TrimSpace(message)
	header, rest, _ := strings.Cut(message, "\n")
	header = strings.TrimSpace(header)

	if m := revertRe.FindStringSubmatch(header); m != nil {
		c.Revert = true
		c.Type = "revert"
		c.Subject = m[1]
	} else if m := headerRe.FindStringSubmatch(header); m != nil {
		c.Type = strings.ToLower(m[headerRe.SubexpIndex("type")])
		c.Scope = strings.TrimSpace(m[headerRe.SubexpIndex("scope")])
		c.Subject = strings.TrimSpace(m[headerRe.SubexpIndex("subject")])
		if m[headerRe.SubexpIndex("bang")] == "!" {
			c.Breaking = true
		}
		if c.Type == "revert" {
			c.Revert = true
		}
	} else {
		c.Subject = header
	}

	body, footers := splitFooters(strings.TrimSpace(rest))
	c.Body = body
	c.Footers = footers
	for _, f := range footers {
		if f.Token == "BREAKING CHANGE" || f.Token == "BREAKING-CHANGE" {
			c.Breaking = true
			c.BreakingNote = f.Value
		}
	}
	if c.Breaking && c.BreakingNote == "" {
		c.BreakingNote = c.Subject
	}

	seen := map[string]bool{}
	addRefs := func(s string) {
		for _, m := range refRe.FindAllStringSubmatch(s, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				c.Refs = append(c.Refs, m[1])
			}
		}
	}
	addRefs(header)
	for _, f := range footers {
		addRefs(f.Token + " #" + f.Value)
	}
	return c
}

// splitFooters separates the trailing footer block from the body. The footer
// block is the last paragraph when its first line looks like a footer;
// continuation lines are folded into the previous footer's value.
func splitFooters(rest string) (string, []Footer) {
	if rest == "" {
		return "", nil
	}
	paras := strings.Split(rest, "\n\n")
	last := paras[len(paras)-1]
	lines := strings.Split(last, "\n")
	if !footerRe.MatchString(lines[0]) {
		return rest, nil
	}
	var footers []Footer
	for _, line := range lines {
		if m := footerRe.FindStringSubmatch(line); m != nil {
			footers = append(footers, Footer{Token: m[1], Value: strings.TrimSpace(m[2])})
			continue
		}
		f := &footers[len(footers)-1]
		f.Value = strings.TrimSpace(f.Value + "\n" + line)
	}
	body := strings.TrimSpace(strings.Join(paras[:len(paras)-1], "\n\n"))
	return body, footers
}
