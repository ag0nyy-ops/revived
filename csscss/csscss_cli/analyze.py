"""Find rulesets that share declarations."""
from __future__ import annotations

import fnmatch
from dataclasses import dataclass

from .parser import Ruleset
from .shorthand import expand_all


@dataclass
class Match:
    a: Ruleset
    b: Ruleset
    shared: list  # declaration keys, sorted

    @property
    def same_selector(self) -> bool:
        return sorted(self.a.selectors) == sorted(self.b.selectors)


def _keys(rs: Ruleset, expand: bool) -> set:
    decls = expand_all(rs.declarations) if expand else rs.declarations
    return {d.key() for d in decls}


def _ignored(rs: Ruleset, patterns: list) -> bool:
    return any(fnmatch.fnmatchcase(s, p) for s in rs.selectors for p in patterns)


def find_matches(rulesets: list, minimum: int = 3, ignore: list = (),
                 expand: bool = True) -> list:
    """Compare rulesets pairwise, only inside the same at-rule context.

    Returns matches sorted by number of shared declarations (descending).
    """
    groups: dict = {}
    for rs in rulesets:
        if _ignored(rs, list(ignore)):
            continue
        groups.setdefault(rs.context, []).append((rs, _keys(rs, expand)))

    matches = []
    for items in groups.values():
        for i in range(len(items)):
            for j in range(i + 1, len(items)):
                shared = items[i][1] & items[j][1]
                if len(shared) >= minimum:
                    matches.append(Match(items[i][0], items[j][0], sorted(shared)))
    matches.sort(key=lambda m: (-len(m.shared), m.a.file, m.a.line, m.b.line))
    return matches
