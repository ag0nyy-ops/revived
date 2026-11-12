package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ag0nyy-ops/revived/chglog/internal/changelog"
	"github.com/ag0nyy-ops/revived/chglog/internal/gitlog"
	"github.com/ag0nyy-ops/revived/chglog/internal/semver"
)

const usage = `chglog - generate a CHANGELOG from Conventional Commits and git tags

Usage:
  chglog [flags]              print the changelog for the repo in the current dir
  chglog -o CHANGELOG.md      write it to a file
  chglog --bump               print the next version implied by unreleased commits
  chglog --latest             notes for the newest release only (for release pages)

Flags:
`

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

type config struct {
	dir, output, format, tagPrefix, tagFilter, nextTag, repoURL, title, remote string
	paths                                                                      stringList
	all, bump, unreleased, latest, noLinks, skipPre, merges, keepReverts, ver  bool
	now                                                                        func() time.Time
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg := config{now: time.Now}
	fs := flag.NewFlagSet("chglog", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	fs.StringVar(&cfg.dir, "C", ".", "run as if started in `dir`")
	fs.StringVar(&cfg.output, "o", "", "write to `file` instead of stdout")
	fs.StringVar(&cfg.format, "format", "md", "output format: md or json")
	fs.StringVar(&cfg.tagPrefix, "tag-prefix", "", "only treat tags with this `prefix` as versions (default: optional \"v\")")
	fs.StringVar(&cfg.tagFilter, "tag-filter", "", "only consider tags matching this `regexp`")
	fs.StringVar(&cfg.nextTag, "next-tag", "", "title unreleased changes as this `tag`; \"auto\" picks it from the commits")
	fs.StringVar(&cfg.repoURL, "repo-url", "", "project `url` for links (default: from the git remote)")
	fs.StringVar(&cfg.remote, "remote", "origin", "git `remote` to read the project url from")
	fs.StringVar(&cfg.title, "title", "Changelog", "document `heading`; empty for none")
	fs.Var(&cfg.paths, "path", "only include commits touching `path` (repeatable, good for monorepos)")
	fs.BoolVar(&cfg.all, "all", false, "include every commit type (docs, chore, ...) and non-conventional commits")
	fs.BoolVar(&cfg.bump, "bump", false, "print the next version and exit")
	fs.BoolVar(&cfg.unreleased, "unreleased", false, "only output unreleased changes")
	fs.BoolVar(&cfg.latest, "latest", false, "only output the newest release")
	fs.BoolVar(&cfg.noLinks, "no-links", false, "don't link commits, issues or compares")
	fs.BoolVar(&cfg.skipPre, "skip-prereleases", false, "fold prerelease tags into the following release")
	fs.BoolVar(&cfg.merges, "merges", false, "include merge commits")
	fs.BoolVar(&cfg.keepReverts, "keep-reverted", false, "keep commits that were reverted within the same release")
	fs.BoolVar(&cfg.ver, "version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q (see -h)", fs.Arg(0))
	}
	if cfg.ver {
		fmt.Fprintln(stdout, "chglog", version)
		return nil
	}
	return generate(cfg, stdout, stderr)
}

func generate(cfg config, stdout, stderr io.Writer) error {
	if cfg.format != "md" && cfg.format != "json" {
		return fmt.Errorf("unknown format %q (want md or json)", cfg.format)
	}
	if cfg.unreleased && cfg.latest {
		return errors.New("--unreleased and --latest can't be combined")
	}
	opts := changelog.Options{
		TagPrefix:        cfg.tagPrefix,
		SkipPrereleases:  cfg.skipPre,
		IncludeMerges:    cfg.merges,
		KeepRevertedPair: cfg.keepReverts,
	}
	if cfg.tagFilter != "" {
		re, err := regexp.Compile(cfg.tagFilter)
		if err != nil {
			return fmt.Errorf("--tag-filter: %w", err)
		}
		opts.TagFilter = re
	}

	repo := gitlog.Repo{Dir: cfg.dir}
	head, err := repo.Head()
	if err != nil {
		return err
	}
	rawTags, err := repo.Tags()
	if err != nil {
		return err
	}
	commits, err := repo.Commits()
	if err != nil {
		return err
	}
	if len(cfg.paths) > 0 {
		if opts.OnlyCommits, err = repo.Touching(cfg.paths); err != nil {
			return err
		}
	}
	tags := changelog.VersionTags(rawTags, opts)
	releases := changelog.Build(commits, tags, head, opts)
	unreleased := &releases[0]

	next := nextVersion(tags, unreleased, cfg.tagPrefix)
	if cfg.bump {
		if changelog.BumpLevel(unreleased.Commits) == semver.None {
			fmt.Fprintln(stderr, "chglog: no release-worthy commits since the last tag")
		}
		fmt.Fprintln(stdout, next)
		return nil
	}
	switch cfg.nextTag {
	case "":
	case "auto":
		unreleased.Tag = next.String()
	default:
		unreleased.Tag = cfg.nextTag
	}
	if unreleased.Tag != "" {
		unreleased.Date = cfg.now().Format("2006-01-02")
	}

	switch {
	case cfg.unreleased:
		releases = releases[:1]
	case cfg.latest:
		releases = releases[1:]
		if len(releases) > 1 {
			releases = releases[:1]
		}
		if len(releases) == 0 || releases[0].Tag == "" {
			return errors.New("no tagged release found")
		}
	}

	var buf bytes.Buffer
	if cfg.format == "json" {
		err = changelog.JSON(&buf, releases)
	} else {
		mo := changelog.MarkdownOptions{Title: cfg.title, Sections: changelog.DefaultSections}
		if cfg.all {
			mo.Sections = changelog.AllSections
		}
		if cfg.unreleased || cfg.latest {
			mo.Title = ""
		}
		if !cfg.noLinks {
			base := cfg.repoURL
			if base == "" {
				base = gitlog.WebURL(repo.RemoteURL(cfg.remote))
			}
			base = strings.TrimSuffix(base, "/")
			mo.Links = gitlog.Links{Base: base, Style: gitlog.DetectStyle(base)}
		}
		err = changelog.Markdown(&buf, releases, mo)
	}
	if err != nil {
		return err
	}
	if cfg.output == "" {
		_, err = stdout.Write(buf.Bytes())
		return err
	}
	return os.WriteFile(cfg.output, buf.Bytes(), 0o644)
}

// nextVersion bumps the newest tag by what the unreleased commits imply.
// With no tags it starts from 0.0.0 using the tag prefix (or "v").
func nextVersion(tags []changelog.VersionTag, unreleased *changelog.Release, prefix string) semver.Version {
	cur := semver.Version{Prefix: prefix}
	if prefix == "" {
		cur.Prefix = "v"
	}
	if n := len(tags); n > 0 {
		cur = tags[n-1].Version
	}
	l := changelog.BumpLevel(unreleased.Commits)
	if l == semver.None {
		return cur
	}
	return semver.Bump(cur, l)
}
