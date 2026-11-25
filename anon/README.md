# anon

A rebuild of [intenthq/anon](https://github.com/intenthq/anon), a UNIX command that anonymises columns of delimited files.

- Original: Go, MIT license, by Intent HQ. Archived, last push April 2019.
- This rebuild: written from scratch, also MIT-compatible, no original code copied.

## Why it died

Intent HQ archived the repository. Open requests (JSON input, k-anonymity, IP addresses) were never finished.

## What's improved

- Go modules and standard library only, no dependencies.
- Header row support, columns by name or 0-based index (fixes #43).
- Columns without a rule pass through unchanged (fixes #42).
- `hash` is HMAC-SHA256 with a salt from the rule, the config or the `ANON_SALT` environment variable. Random per-run salts are gone, so output is joinable across runs (fixes #41).
- Date tokens are `YYYY MM DD hh mm ss MMM MMMM`, and errors give the line and column (fixes #44).
- New actions: `mask`, `redact`, `ip`, `date` (reformat) on top of `nothing`, `hash`, `outcode`, `year`, `range`.
- Unknown config keys are rejected, so typos are caught.
- Streaming, with deterministic FNV sampling on any column.

## Usage

```sh
go build -o anon .
ANON_SALT=secret ./anon -c examples/config.json -i examples/people.csv -o out.csv
./anon -c config.json < in.csv > out.csv
```

Flags: `-c` config (default `config.json`), `-i` input, `-o` output, `-salt-env` name of the salt variable.

Config:

```json
{
  "delimiter": ",",
  "header": true,
  "salt": "optional",
  "sampling": { "mod": 10, "column": "id" },
  "rules": [{ "column": "email", "action": "hash" }]
}
```

| action | options | result |
|---|---|---|
| `nothing` | | unchanged |
| `hash` | `salt` | hex HMAC-SHA256 |
| `outcode` | | UK postcode outward part |
| `year` | `format` | year of a date |
| `date` | `format`, `outputFormat` | reformatted date |
| `range` | `ranges` with `gt/gte/lt/lte`, `output` | bucket label |
| `mask` | `keep`, `char` | all but the last `keep` characters masked |
| `redact` | `char` | fixed replacement text |
| `ip` | | IPv4 /24, IPv6 /48 |

Hashing and masking reduce but do not eliminate re-identification risk. Do not treat the output as safe to publish without review.

## Tests

```sh
go test ./...
```
