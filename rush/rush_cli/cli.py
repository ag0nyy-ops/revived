"""Command line entry point."""
from __future__ import annotations

import argparse
import os
import sys
from typing import List, Optional

from . import __version__
from .discover import find, read
from .graph import order
from .parser import RushfileError, parse
from .runner import run_task


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="rush", description="Run bash snippets from a rushfile.yml")
    p.add_argument("tasks", nargs="*", help="task names to run")
    p.add_argument("-a", "--all", action="store_true", help="run every enabled task")
    p.add_argument("-l", "--list", action="store_true", help="list tasks")
    p.add_argument("-v", "--view", action="store_true", help="with --list, show commands")
    p.add_argument("-n", "--dry-run", action="store_true", help="print the run order only")
    p.add_argument("--no-deps", action="store_true", help="skip @dependencies")
    p.add_argument("--ignore-errors", action="store_true", help="keep going after a failure")
    p.add_argument("--hide-outputs", action="store_true", help="discard task output")
    p.add_argument("-p", "--path", action="store_true", help="print the rushfile path")
    p.add_argument("-f", "--file", help="use this rushfile")
    p.add_argument("-V", "--version", action="version", version=f"rush {__version__}")
    return p


def main(argv: Optional[List[str]] = None) -> int:
    args = build_parser().parse_args(argv)
    path = args.file or find(os.getcwd())
    if not path or not os.path.isfile(path):
        print("rush: no rushfile.yml found", file=sys.stderr)
        return 2
    if args.path:
        print(os.path.abspath(path))
        return 0
    try:
        tasks = parse(read(path))
        if args.list or (not args.tasks and not args.all):
            for t in tasks.values():
                mark = " (disabled)" if t.disabled else ""
                print(f"{t.name}{mark}")
                if args.view:
                    for c in t.commands:
                        print(f"    {c}")
            return 0
        wanted = [n for n, t in tasks.items() if not t.disabled] if args.all else args.tasks
        plan = order(tasks, wanted, with_deps=not args.no_deps)
    except RushfileError as e:
        print(f"rush: {e}", file=sys.stderr)
        return 2
    if args.dry_run:
        print("\n".join(plan))
        return 0
    cwd = os.path.dirname(os.path.abspath(path))
    status = 0
    for name in plan:
        print(f"==> {name}", file=sys.stderr)
        rc = run_task(tasks[name], cwd=cwd, quiet=args.hide_outputs)
        if rc != 0:
            print(f"rush: task {name!r} failed with status {rc}", file=sys.stderr)
            status = rc
            if not args.ignore_errors:
                return rc
    return status


if __name__ == "__main__":
    sys.exit(main())
