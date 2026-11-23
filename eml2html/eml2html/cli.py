import argparse
import sys
from pathlib import Path

from . import __version__
from .convert import convert, parse_bytes


def collect(paths, recursive):
    """Yield (path, error) pairs; error is None for usable inputs."""
    for raw in paths:
        p = Path(raw)
        if p.is_dir():
            it = p.rglob("*.eml") if recursive else p.glob("*.eml")
            for f in sorted(it):
                yield f, None
        elif not p.exists():
            yield p, "does not exist"
        elif not p.is_file():
            yield p, "is not a file"
        elif p.suffix.lower() != ".eml":
            yield p, "is not an .eml file"
        else:
            yield p, None


def build_parser():
    ap = argparse.ArgumentParser(prog="eml2html",
                                 description="Convert .eml files to HTML.")
    ap.add_argument("inputs", nargs="+", help=".eml files or directories")
    ap.add_argument("-o", "--out-dir", help="write output here (default: next to input)")
    ap.add_argument("-f", "--force", action="store_true", help="overwrite existing files")
    ap.add_argument("-r", "--recursive", action="store_true", help="recurse into directories")
    ap.add_argument("--headers", action="store_true", help="include a From/To/Date/Subject table")
    ap.add_argument("--stdout", action="store_true", help="print HTML instead of writing files")
    ap.add_argument("--version", action="version", version=__version__)
    return ap


def main(argv=None):
    args = build_parser().parse_args(argv)
    failures = 0
    for path, err in collect(args.inputs, args.recursive):
        if err:
            print("skipping %s: %s" % (path, err), file=sys.stderr)
            failures += 1
            continue
        try:
            out = convert(parse_bytes(path.read_bytes()), args.headers)
        except OSError as e:
            print("error: %s: %s" % (path, e), file=sys.stderr)
            failures += 1
            continue
        if args.stdout:
            sys.stdout.write(out)
            continue
        dest_dir = Path(args.out_dir) if args.out_dir else path.parent
        dest_dir.mkdir(parents=True, exist_ok=True)
        dest = dest_dir / (path.stem + ".html")
        if dest.exists() and not args.force:
            print("skipping %s: %s exists (use --force)" % (path, dest), file=sys.stderr)
            failures += 1
            continue
        dest.write_text(out, encoding="utf-8")
        print("wrote %s" % dest)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
