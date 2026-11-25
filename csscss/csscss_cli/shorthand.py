"""Expand box shorthands so `margin: 0 auto` matches `margin-left: auto`."""
from __future__ import annotations

from .parser import Declaration

SIDES = ("top", "right", "bottom", "left")

# shorthand -> pattern for the longhand names ("{}" is replaced by the side)
BOX = {
    "margin": "margin-{}",
    "padding": "padding-{}",
    "border-width": "border-{}-width",
    "border-style": "border-{}-style",
    "border-color": "border-{}-color",
}


def _values(value: str) -> list:
    out, cur, depth = [], [], 0
    for ch in value:
        if ch == "(":
            depth += 1
        elif ch == ")":
            depth = max(0, depth - 1)
        if ch.isspace() and depth == 0:
            if cur:
                out.append("".join(cur))
                cur = []
        else:
            cur.append(ch)
    if cur:
        out.append("".join(cur))
    return out


def expand(decl: Declaration) -> list:
    pattern = BOX.get(decl.prop)
    if pattern is None:
        return [decl]
    vals = _values(decl.value)
    if not 1 <= len(vals) <= 4 or any(v in ("inherit", "initial", "unset") for v in vals) and len(vals) > 1:
        return [decl]
    top = vals[0]
    right = vals[1] if len(vals) > 1 else top
    bottom = vals[2] if len(vals) > 2 else top
    left = vals[3] if len(vals) > 3 else right
    return [
        Declaration(pattern.format(side), v, decl.important)
        for side, v in zip(SIDES, (top, right, bottom, left))
    ]


def expand_all(decls: list) -> list:
    out = []
    for d in decls:
        out.extend(expand(d))
    return out
