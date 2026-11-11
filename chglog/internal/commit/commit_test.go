package commit

import (
	"reflect"
	"testing"
)

func TestParseHeader(t *testing.T) {
	tests := []struct {
		msg                    string
		typ, scope, subject    string
		breaking, conventional bool
	}{
		{"feat: add thing", "feat", "", "add thing", false, true},
		{"Fix(Parser): handle tabs", "fix", "Parser", "handle tabs", false, true},
		{"feat(api)!: drop v1", "feat", "api", "drop v1", true, true},
		{"refactor!: rewrite core", "refactor", "", "rewrite core", true, true},
		{"build(deps-dev): bump x", "build", "deps-dev", "bump x", false, true},
		{"Update README", "", "", "Update README", false, false},
		{"feat:missing space", "", "", "feat:missing space", false, false},
		{"feat(): empty scope", "feat", "", "empty scope", false, true},
	}
	for _, tt := range tests {
		c := Parse(Commit{}, tt.msg)
		if c.Type != tt.typ || c.Scope != tt.scope || c.Subject != tt.subject ||
			c.Breaking != tt.breaking || c.Conventional() != tt.conventional {
			t.Errorf("Parse(%q) = type %q scope %q subject %q breaking %v; want %q %q %q %v",
				tt.msg, c.Type, c.Scope, c.Subject, c.Breaking, tt.typ, tt.scope, tt.subject, tt.breaking)
		}
	}
}

func TestParseBodyAndFooters(t *testing.T) {
	msg := "fix(auth): refresh tokens early\r\n\r\nTokens expired mid-request.\r\n\r\nSecond paragraph.\r\n\r\nBREAKING CHANGE: sessions now last 1h\r\n  instead of 24h\r\nReviewed-by: Sam\r\nCloses #42\r\nRefs #7, #42\r\n"
	c := Parse(Commit{}, msg)
	if !c.Breaking {
		t.Fatal("BREAKING CHANGE footer should mark the commit breaking")
	}
	if want := "sessions now last 1h\ninstead of 24h"; c.BreakingNote != want {
		t.Errorf("BreakingNote = %q, want %q", c.BreakingNote, want)
	}
	if want := "Tokens expired mid-request.\n\nSecond paragraph."; c.Body != want {
		t.Errorf("Body = %q, want %q", c.Body, want)
	}
	if len(c.Footers) != 4 || c.Footers[1].Token != "Reviewed-by" || c.Footers[2].Token != "Closes" {
		t.Errorf("Footers = %+v", c.Footers)
	}
	if want := []string{"42", "7"}; !reflect.DeepEqual(c.Refs, want) {
		t.Errorf("Refs = %v, want %v", c.Refs, want)
	}
}

func TestBangUsesSubjectAsNote(t *testing.T) {
	c := Parse(Commit{}, "feat!: remove legacy flags")
	if c.BreakingNote != "remove legacy flags" {
		t.Errorf("BreakingNote = %q", c.BreakingNote)
	}
}

func TestBodyIsNotMistakenForFooters(t *testing.T) {
	c := Parse(Commit{}, "docs: explain\n\nThis is just prose.\nNothing to see.")
	if len(c.Footers) != 0 || c.Body != "This is just prose.\nNothing to see." {
		t.Errorf("body %q footers %+v", c.Body, c.Footers)
	}
}

func TestRefsInHeader(t *testing.T) {
	c := Parse(Commit{}, "fix: crash on empty input (#12)")
	if !reflect.DeepEqual(c.Refs, []string{"12"}) {
		t.Errorf("Refs = %v", c.Refs)
	}
}

func TestRevert(t *testing.T) {
	c := Parse(Commit{}, "Revert \"feat: shiny\"\n\nThis reverts commit 0123456789abcdef0123456789abcdef01234567.\n")
	if !c.Revert || c.Type != "revert" || c.Subject != "feat: shiny" {
		t.Errorf("got %+v", c)
	}
	if c.RevertsHash != "0123456789abcdef0123456789abcdef01234567" {
		t.Errorf("RevertsHash = %q", c.RevertsHash)
	}
	c = Parse(Commit{}, "revert: feat: shiny")
	if !c.Revert || c.Subject != "feat: shiny" {
		t.Errorf("conventional revert: %+v", c)
	}
}

func TestIsMerge(t *testing.T) {
	if (Commit{Parents: []string{"a"}}).IsMerge() || !(Commit{Parents: []string{"a", "b"}}).IsMerge() {
		t.Error("IsMerge wrong")
	}
}
