"""Parser for the small rushfile subset: top-level `name: |` blocks."""
from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Dict, List

KEY_RE = re.compile(r"^(#)?([A-Za-z0-9_][A-Za-z0-9_.-]*):[ \t]*(.*)$")
BLOCK_MARKERS = {"|", "|-", "|+", ">", ">-"}


class RushfileError(Exception):
    """Raised for malformed rushfiles; message includes the line number."""


@dataclass
class Task:
    name: str
    lines: List[str] = field(default_factory=list)
    disabled: bool = False
    line_no: int = 0

    @property
    def commands(self) -> List[str]:
        """Command lines, without dependency markers or comments."""
        out = []
        for ln in self.lines:
            s = ln.strip()
            if not s or s.startswith("#") or s.startswith("@"):
                continue
            out.append(ln)
        return out

    @property
    def deps(self) -> List[str]:
        return [ln.strip()[1:].strip() for ln in self.lines
                if ln.strip().startswith("@") and ln.strip()[1:].strip()]

    @property
    def script(self) -> str:
        return "\n".join(self.commands) + "\n"


def _dedent(lines: List[str]) -> List[str]:
    indents = [len(l) - len(l.lstrip()) for l in lines if l.strip()]
    cut = min(indents) if indents else 0
    return [l[cut:].rstrip() if l.strip() else "" for l in lines]


def parse(text: str) -> Dict[str, Task]:
    tasks: Dict[str, Task] = {}
    current = None
    body: List[str] = []

    def flush():
        if current is not None:
            current.lines = _dedent(body)

    for no, raw in enumerate(text.splitlines(), 1):
        indented = raw[:1] in (" ", "\t")
        if not raw.strip():
            if current is not None:
                body.append("")
            continue
        if not indented:
            if raw.lstrip().startswith("#") and not KEY_RE.match(raw):
                continue
            m = KEY_RE.match(raw)
            if not m:
                raise RushfileError(f"line {no}: expected 'name: |', got {raw!r}")
            flush()
            disabled, name, rest = bool(m.group(1)), m.group(2), m.group(3).strip()
            if name in tasks:
                raise RushfileError(f"line {no}: duplicate task {name!r}")
            current = Task(name=name, disabled=disabled, line_no=no)
            tasks[name] = current
            body = []
            if rest in BLOCK_MARKERS:
                pass
            elif rest:
                body.append(rest)
            else:
                raise RushfileError(
                    f"line {no}: task {name!r} has no block marker; write '{name}: |'")
        else:
            if current is None:
                raise RushfileError(f"line {no}: indented text before any task")
            body.append(raw)
    flush()
    return tasks
