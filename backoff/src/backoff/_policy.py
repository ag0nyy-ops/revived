"""Shared decision logic used by both the sync and async runners."""
from __future__ import annotations

import time
from dataclasses import dataclass
from typing import Any, Callable, Iterator, Optional, Sequence

from . import jitter as _jitter
from .details import Details

Handler = Callable[[Details], Any]

# Global switch: when False every decorated function is simply called once.
enabled = True


@dataclass
class Policy:
    wait_gen: Callable[[], Iterator[float]]
    max_tries: Optional[int] = None
    max_time: Optional[float] = None
    jitter: Optional[Callable[[float], float]] = _jitter.full_jitter
    clock: Callable[[], float] = time.monotonic
    on_success: Sequence[Handler] = ()
    on_backoff: Sequence[Handler] = ()
    on_giveup: Sequence[Handler] = ()

    def next_wait(self, gen: Iterator[float], elapsed: float, tries: int) -> Optional[float]:
        """Return seconds to sleep, or None when the call must give up."""
        if self.max_tries is not None and tries >= self.max_tries:
            return None
        wait = next(gen)
        if self.jitter is not None:
            wait = self.jitter(wait)
        if self.max_time is not None:
            remaining = self.max_time - elapsed
            if remaining <= 0 or wait >= remaining:
                return None
        return wait


def fire(handlers: Sequence[Handler], details: Details) -> None:
    for handler in handlers:
        handler(details)
