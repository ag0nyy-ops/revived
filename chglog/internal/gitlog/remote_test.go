package gitlog

import "testing"

func TestWebURL(t *testing.T) {
	tests := map[string]string{
		"git@github.com:acme/widget.git":                     "https://github.com/acme/widget",
		"git@github.com:acme/widget":                         "https://github.com/acme/widget",
		"https://github.com/acme/widget.git":                 "https://github.com/acme/widget",
		"https://user:token@github.com/acme/widget.git":      "https://github.com/acme/widget",
		"ssh://git@gitlab.example.com:2222/grp/sub/proj.git": "https://gitlab.example.com/grp/sub/proj",
		"git@bitbucket.org:team/repo.git":                    "https://bitbucket.org/team/repo",
		"/srv/git/repo.git":                                  "",
		"file:///srv/git/repo.git":                           "",
		"../other":                                           "",
		"":                                                   "",
	}
	for in, want := range tests {
		if got := WebURL(in); got != want {
			t.Errorf("WebURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinks(t *testing.T) {
	gh := Links{Base: "https://github.com/a/b", Style: DetectStyle("https://github.com/a/b")}
	gl := Links{Base: "https://gitlab.com/a/b", Style: DetectStyle("https://gitlab.com/a/b")}
	bb := Links{Base: "https://bitbucket.org/a/b", Style: DetectStyle("https://bitbucket.org/a/b")}
	checks := []struct{ got, want string }{
		{gh.Commit("abc"), "https://github.com/a/b/commit/abc"},
		{gh.Compare("v1", "v2"), "https://github.com/a/b/compare/v1...v2"},
		{gh.Issue("3"), "https://github.com/a/b/issues/3"},
		{gl.Commit("abc"), "https://gitlab.com/a/b/-/commit/abc"},
		{gl.Compare("v1", "v2"), "https://gitlab.com/a/b/-/compare/v1...v2"},
		{gl.Issue("3"), "https://gitlab.com/a/b/-/issues/3"},
		{bb.Commit("abc"), "https://bitbucket.org/a/b/commits/abc"},
		{gh.Compare("", "v1"), ""},
		{Links{}.Commit("abc"), ""},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
