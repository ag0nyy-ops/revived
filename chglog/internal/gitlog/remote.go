package gitlog

import (
	"net/url"
	"regexp"
	"strings"
)

var scpLike = regexp.MustCompile(`^(?:[\w.-]+@)?([\w.-]+):(?:/)?(.+?)(?:\.git)?/?$`)

// WebURL turns a git remote URL (https, ssh:// or scp-like git@host:owner/repo)
// into the https URL of the project page. It returns "" for local paths and
// anything it does not recognise. Credentials are never kept.
func WebURL(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil || u.Host == "" {
			return ""
		}
		switch u.Scheme {
		case "http", "https", "ssh", "git", "git+ssh":
		default:
			return ""
		}
		path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
		if path == "" {
			return ""
		}
		return "https://" + u.Hostname() + "/" + path
	}
	if m := scpLike.FindStringSubmatch(remote); m != nil && strings.Contains(m[1], ".") {
		return "https://" + m[1] + "/" + strings.Trim(m[2], "/")
	}
	return ""
}

// Links builds commit, compare and issue URLs for a hosting provider.
type Links struct {
	Base  string // project page, e.g. https://github.com/owner/repo
	Style string // "github" (also Gitea/Codeberg), "gitlab" or "bitbucket"
}

// DetectStyle guesses the hosting style from the host name.
func DetectStyle(base string) string {
	host := base
	if u, err := url.Parse(base); err == nil {
		host = u.Hostname()
	}
	switch {
	case strings.Contains(host, "gitlab"):
		return "gitlab"
	case strings.Contains(host, "bitbucket"):
		return "bitbucket"
	}
	return "github"
}

func (l Links) ok() bool { return l.Base != "" }

// Commit returns the URL of a commit, or "".
func (l Links) Commit(hash string) string {
	if !l.ok() {
		return ""
	}
	switch l.Style {
	case "gitlab":
		return l.Base + "/-/commit/" + hash
	case "bitbucket":
		return l.Base + "/commits/" + hash
	}
	return l.Base + "/commit/" + hash
}

// Compare returns the URL comparing two refs, or "".
func (l Links) Compare(from, to string) string {
	if !l.ok() || from == "" || to == "" {
		return ""
	}
	switch l.Style {
	case "gitlab":
		return l.Base + "/-/compare/" + from + "..." + to
	case "bitbucket":
		return l.Base + "/branches/compare/" + to + "%0D" + from
	}
	return l.Base + "/compare/" + from + "..." + to
}

// Issue returns the URL of an issue or pull request number, or "".
func (l Links) Issue(n string) string {
	if !l.ok() {
		return ""
	}
	switch l.Style {
	case "gitlab":
		return l.Base + "/-/issues/" + n
	case "bitbucket":
		return l.Base + "/issues/" + n
	}
	return l.Base + "/issues/" + n
}
