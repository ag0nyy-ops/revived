"""Command line entry point."""
from __future__ import annotations

import argparse
import sys

from . import __version__
from .analyze import find_matches
from .parser import CssError, parse
from .report import to_json, to_text


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="csscss",
        description="Find CSS rulesets that repeat the same declarations.",
    )
    p.add_argument("files", nargs="+", help="CSS files, or - for stdin")
    p.add_argument("-n", "--min", type=int, default=3, metavar="N",
                   help="only report rulesets sharing at least N declarations (default 3)")
    p.add_argument("-v", "--verbose", action="store_true", help="list the shared declarations")
    p.add_argument("--json", action="store_true", help="machine-readable output")
    p.add_argument("-i", "--ignore", action="append", default=[], metavar="GLOB",
                   help="skip rulesets with a selector matching GLOB (repeatable)")
    p.add_argument("--no-expand", action="store_true",
                   help="do not expand margin/padding/border shorthands")
    p.add_argument("--fail-on-duplicates", action="store_true",
                   help="exit with status 1 when duplicates are found")
    p.add_argument("--version", action="version", version=f"csscss {__version__}")
    return p


def main(argv=None) -> int:
    args = build_parser().parse_args(argv)
    if args.min < 1:
        print("csscss: --min must be at least 1", file=sys.stderr)
        return 2
    rulesets = []
    for path in args.files:
        try:
            if path == "-":
                text, name = sys.stdin.read(), "<stdin>"
            else:
                with open(path, encoding="utf-8-sig", errors="replace") as f:
                    text, name = f.read(), path
            rulesets.extend(parse(text, name))
        except OSError as e:
            print(f"csscss: {e.filename or path}: {e.strerror}", file=sys.stderr)
            return 2
        except CssError as e:
            print(f"csscss: {path}: {e}", file=sys.stderr)
            return 2
    matches = find_matches(rulesets, args.min, args.ignore, not args.no_expand)
    if args.json:
        print(to_json(matches))
    elif matches:
        print(to_text(matches, args.verbose))
    return 1 if matches and args.fail_on_duplicates else 0


if __name__ == "__main__":
    sys.exit(main())
