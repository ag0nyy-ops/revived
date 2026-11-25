# joblint

A rebuild of [rowanmanning/joblint](https://github.com/rowanmanning/joblint), a linter for tech job posts.

- Original: rowanmanning/joblint (JavaScript, MIT, archived March 2023)
- This rebuild: written from scratch, MIT. Credit for the idea goes to Rowan Manning. No original source or rule lists were copied.

## Why the original died

The author marked it as a 2013 side project that was "very naive" in how it matched text and what it flagged, and archived it. It depended on old Node and npm packages and only handled one input at a time.

## What's improved

- Zero dependencies, ESM, needs Node 20+ (fixes #53)
- Whole-word, case-insensitive matching with phrase support, so "bro" does not fire inside "broadcaster" (fixes #54)
- Every finding has a line and column (fixes #55)
- Several files plus stdin in one run
- JSON config to turn rules off, change levels, or add your own terms or regexes (fixes #56)
- `--fail-on error|warning|notice|none` and exit code 2 for usage and input errors
- Typographic quotes and dashes are normalised before matching
- Text and JSON reporters, `--list-rules`

## Usage

```sh
node bin/joblint.js post.txt other.txt
cat post.txt | node bin/joblint.js --fail-on warning
node bin/joblint.js -f json --pretty post.txt
node bin/joblint.js --list-rules
```

Example:

```
$ printf 'Rockstar wanted, ASAP\n10+ years. Salesman welcome.\n' | joblint
<stdin>:1:1  warning Hype job title; say what the role actually is (hype-title: "Rockstar")
<stdin>:1:18  warning Pressure wording; candidates read it as disorganisation (urgent: "ASAP")
<stdin>:2:1  warning Very high years-of-experience requirement (years-experience: "10+ years")
<stdin>:2:12  error   Gendered job title or term; use a neutral alternative (gendered-role: "Salesman")
4 issues: 1 error, 3 warning, 0 notice
```

### Config

```json
{
  "rules": { "hype-title": "off", "urgent": "error" },
  "custom": [
    { "id": "no-php", "level": "error", "message": "No PHP", "terms": ["php"] },
    { "id": "salary-range", "level": "notice", "message": "Mentions k", "pattern": "\\b\\d+k\\b" }
  ]
}
```

Run with `joblint -c joblint.json post.txt`.

### Exit codes

`0` nothing at or above `--fail-on` (default `error`), `1` findings at that level, `2` bad usage, config or unreadable input.

## Limits

This is a word list, not a judge. Treat findings as prompts to reread the post, and tune the rules for your team.

## Tests

```sh
npm test
```
