"""Dependency ordering for tasks."""
from __future__ import annotations

from typing import Dict, List

from .parser import RushfileError, Task


def order(tasks: Dict[str, Task], wanted: List[str], with_deps: bool = True) -> List[str]:
    """Return task names in run order. Each task appears once."""
    result: List[str] = []
    state: Dict[str, int] = {}

    def visit(name: str, chain: List[str]) -> None:
        if name not in tasks:
            where = f" (needed by {chain[-1]!r})" if chain else ""
            raise RushfileError(f"unknown task {name!r}{where}")
        if tasks[name].disabled:
            raise RushfileError(f"task {name!r} is disabled")
        if state.get(name) == 2:
            return
        if state.get(name) == 1:
            cycle = " -> ".join(chain[chain.index(name):] + [name])
            raise RushfileError(f"dependency cycle: {cycle}")
        state[name] = 1
        if with_deps:
            for dep in tasks[name].deps:
                visit(dep, chain + [name])
        state[name] = 2
        result.append(name)

    for w in wanted:
        visit(w, [])
    return result
