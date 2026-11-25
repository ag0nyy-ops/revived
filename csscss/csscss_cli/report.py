"""Render matches as text or JSON."""
from __future__ import annotations

import json


def _where(rs) -> str:
    sel = "{" + rs.label + "}"
    loc = f"{rs.file}:{rs.line}" if rs.file else f"line {rs.line}"
    ctx = f" in {' '.join(rs.context)}" if rs.context else ""
    return f"{sel} ({loc}){ctx}"


def to_text(matches: list, verbose: bool = False) -> str:
    lines = []
    for m in matches:
        n = m.count
        note = " [same selector]" if m.same_selector else ""
        lines.append(f"{_where(m.a)} and {_where(m.b)} share {n} rule{'s' if n != 1 else ''}{note}")
        if verbose:
            lines.extend(f"    {k}" for k in m.shared)
    return "\n".join(lines)


def _rs(rs) -> dict:
    return {"selectors": rs.selectors, "file": rs.file, "line": rs.line,
            "context": list(rs.context)}


def to_json(matches: list) -> str:
    data = [{"a": _rs(m.a), "b": _rs(m.b), "same_selector": m.same_selector,
             "count": m.count, "shared": m.shared} for m in matches]
    return json.dumps(data, indent=2)
