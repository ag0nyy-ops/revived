# fixjson

A rebuild of [SunGard-Labs/fix2json](https://github.com/SunGard-Labs/fix2json) (JavaScript, MIT license, archived in 2023). Credit for the idea and the original tool goes to SunGard Labs. This version is written from scratch and shares no code with it. The bundled `dict/FIX44.xml` is a small dictionary written for this tool.

## Why the original died

SunGard announced it would archive the repo on 1 June 2023 and pointed people to Google's Market Data Transcoder. Several issues stayed open: a dictionary was mandatory and needed a full path, output depended on attribute position in the dictionary XML, checksum could not be dropped, there was no header/body/trailer split, and JSON could not be turned back into FIX.

## What's improved

- Zero dependencies, ESM, Node 20+, with an own small XML reader for QuickFIX-style dictionaries (#24)
- Dictionary is optional. Without one you get raw tag numbers. `--dict FIX44` selects a bundled one by name, and a path works too (#23)
- Dictionaries are read by attribute name, not position (#24)
- Output is split into `Header`, `Body` and `Trailer`, or `--flat` (#25)
- `--no-checksum` drops CheckSum(10)
- `--validate` checks BodyLength and CheckSum and reports short repeating groups
- `--reverse` turns JSON back into FIX, recomputing length and checksum (#26)
- JSON, JSON lines or YAML from one command, `--enums` for readable enum values
- SOH, `|` and `^A` delimiters detected automatically, `.gz` input unpacked

## Usage

```bash
node bin/fixjson.js -d FIX44 messages.fix                # JSON
node bin/fixjson.js -d FIX44 -e -f yaml messages.fix     # YAML with enum names
cat messages.fix | node bin/fixjson.js -f jsonl --flat   # no dictionary
node bin/fixjson.js -d FIX44 --validate messages.fix     # exit 1 on bad checksum
node bin/fixjson.js -d FIX44 -r --delim '|' out.json     # JSON back to FIX
npm test
```

Repeating groups appear under their `No...` field name as arrays. Custom dictionaries use the QuickFIX XML layout (`fields`, `header`, `trailer`, `messages`, `components`). The bundled FIX44 file is only a subset, so bring your own dictionary for other messages.

## License

MIT. Original copyright SunGard Labs.
