"""Call details handed to event handlers."""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Callable, Dict, Optional, Tuple


@dataclass(frozen=True)
class Details:
    target: Callable[..., Any]
    args: Tuple[Any, ...]
    kwargs: Dict[str, Any]
    tries: int
    elapsed: float
    wait: Optional[float] = None
    value: Any = None
    exception: Optional[BaseException] = field(default=None, repr=False)
