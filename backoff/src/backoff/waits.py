"""Wait generators. Each yields the number of seconds to sleep before the next try."""
from __future__ import annotations

from typing import Iterator, Optional


def expo(base: float = 2, factor: float = 1, max_value: Optional[float] = None) -> Iterator[float]:
    """factor * base**n for n = 0, 1, 2, ... capped at max_value."""
    n = 0
    while True:
        value = factor * base**n
        if max_value is not None and value >= max_value:
            while True:
                yield max_value
        yield value
        n += 1


def fibo(max_value: Optional[float] = None) -> Iterator[float]:
    """1, 1, 2, 3, 5, 8, ... capped at max_value."""
    a, b = 1, 1
    while True:
        if max_value is not None and a >= max_value:
            while True:
                yield max_value
        yield a
        a, b = b, a + b


def constant(interval: float = 1) -> Iterator[float]:
    """The same wait forever."""
    while True:
        yield interval
