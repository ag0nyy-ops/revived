"""A small CSS parser that yields rulesets with their at-rule context.

It only understands what redundancy analysis needs: style rules, nested
at-rules (@media, @supports, ...), comments, strings and url() values.
"""
from __future__ import annotations

from dataclasses import dataclass


class CssError(ValueError):
    def __init__(self, message: str, line: int):
        super().__init__(f"line {line}: {message}")
        self.line = line


@dataclass
class Declaration:
    prop: str
    value: str
    important: bool = False

    def key(self) -> str:
        return f"{self.prop}:{self.value}" + ("!important" if self.important else "")


@dataclass
class Ruleset:
    selectors: list
    declarations: list
    line: int
    context: tuple = ()
    file: str = ""

    @property
    def label(self) -> str:
        return ", ".join(self.selectors)


def strip_comments(text: str) -> str:
    """Replace comments with spaces, keeping newlines so line numbers survive."""
    out = []
    i, n = 0, len(text)
    quote = ""
    while i < n:
        c = text[i]
        if quote:
            out.append(c)
            if c == "\\" and i + 1 < n:
                out.append(text[i + 1])
                i += 1
            elif c == quote or c == "\n":
                quote = ""
        elif c in "\"'":
            quote = c
            out.append(c)
        elif text.startswith("/*", i):
            end = text.find("*/", i + 2)
            end = n if end == -1 else end + 2
            out.append("".join("\n" if ch == "\n" else " " for ch in text[i:end]))
            i = end
            continue
        else:
            out.append(c)
        i += 1
    return "".join(out)


def _normalize_ws(s: str) -> str:
    return " ".join(s.split())


def _split_top(text: str, sep: str) -> list:
    """Split on sep outside of strings, parentheses and brackets."""
    parts, depth, quote, cur = [], 0, "", []
    i = 0
    while i < len(text):
        c = text[i]
        if quote:
            cur.append(c)
            if c == "\\" and i + 1 < len(text):
                cur.append(text[i + 1])
                i += 1
            elif c == quote:
                quote = ""
        elif c in "\"'":
            quote = c
            cur.append(c)
        elif c in "([":
            depth += 1
            cur.append(c)
        elif c in ")]":
            depth = max(0, depth - 1)
            cur.append(c)
        elif c == sep and depth == 0:
            parts.append("".join(cur))
            cur = []
        else:
            cur.append(c)
        i += 1
    parts.append("".join(cur))
    return parts


def parse_declarations(body: str) -> list:
    decls = []
    for chunk in _split_top(body, ";"):
        chunk = chunk.strip()
        if not chunk or ":" not in chunk:
            continue
        prop, value = chunk.split(":", 1)
        prop = prop.strip().lower()
        value = _normalize_ws(value)
        important = False
        low = value.lower()
        if low.endswith("!important"):
            important = True
            value = value[: -len("!important")].rstrip()
        elif low.endswith("! important"):
            important = True
            value = value[: -len("! important")].rstrip()
        if prop and value:
            decls.append(Declaration(prop, value, important))
    return decls


def parse(text: str, file: str = "") -> list:
    """Parse CSS text into a flat list of Rulesets."""
    text = strip_comments(text)
    rulesets: list = []
    _parse_block(text, 0, len(text), (), file, rulesets)
    return rulesets


def _line(text: str, pos: int) -> int:
    return text.count("\n", 0, pos) + 1


def _find_block_end(text: str, start: int, end: int) -> int:
    """start is just after '{'. Return index of the matching '}'."""
    depth, quote = 1, ""
    i = start
    while i < end:
        c = text[i]
        if quote:
            if c == "\\":
                i += 1
            elif c == quote:
                quote = ""
        elif c in "\"'":
            quote = c
        elif c == "{":
            depth += 1
        elif c == "}":
            depth -= 1
            if depth == 0:
                return i
        i += 1
    raise CssError("unterminated block", _line(text, start))


def _parse_block(text, start, end, context, file, out):
    i = start
    while i < end:
        # skip whitespace
        while i < end and text[i].isspace():
            i += 1
        if i >= end:
            break
        # prelude runs until '{' or ';' (at-statement) outside strings/parens
        j, depth, quote = i, 0, ""
        while j < end:
            c = text[j]
            if quote:
                if c == "\\":
                    j += 1
                elif c == quote:
                    quote = ""
            elif c in "\"'":
                quote = c
            elif c == "(":
                depth += 1
            elif c == ")":
                depth = max(0, depth - 1)
            elif depth == 0 and c in "{;}":
                break
            j += 1
        if j >= end or text[j] == "}":
            if j < end:
                raise CssError("unexpected '}'", _line(text, j))
            break  # trailing junk without a block
        prelude = _normalize_ws(text[i:j])
        if text[j] == ";":  # @charset, @import, ...
            i = j + 1
            continue
        close = _find_block_end(text, j + 1, end)
        line = _line(text, i + (len(text[i:j]) - len(text[i:j].lstrip())))
        if prelude.startswith("@"):
            name = prelude.split(None, 1)[0].lower()
            if name in ("@media", "@supports", "@document", "@layer", "@container"):
                _parse_block(text, j + 1, close, context + (prelude,), file, out)
            # @font-face, @keyframes, @page etc. are not selector rulesets
        else:
            selectors = [_normalize_ws(s) for s in _split_top(prelude, ",") if s.strip()]
            decls = parse_declarations(text[j + 1 : close])
            if selectors:
                out.append(Ruleset(selectors, decls, line, context, file))
        i = close + 1
