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
    count: int = 0  # shared declarations as written (a shorthand counts once)

    @property
    def same_selector(self) -> bool:
        return sorted(self.a.selectors) == sorted(self.b.selectors)


def _keys(rs: Ruleset, expand: bool) -> dict:
    """Map each comparable declaration key to the property it was written as."""
    if expand:
        return {d.key(): src for src, d in expand_all(rs.declarations)}
    return {d.key(): d.prop for d in rs.declarations}


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
                shared = items[i][1].keys() & items[j][1].keys()
                count = len({items[i][1][k] for k in shared})
                if count >= minimum:
                    matches.append(Match(items[i][0], items[j][0], sorted(shared), count))
    matches.sort(key=lambda m: (-m.count, m.a.file, m.a.line, m.b.line))
    return matches
