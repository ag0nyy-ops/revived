"""Typed retry decorators with backoff for sync and async functions."""
from __future__ import annotations

import time
from typing import Any, Callable, Iterator, Optional, Sequence, Tuple, Type, Union

from . import _decorate, _policy
from .details import Details
from .jitter import equal_jitter, full_jitter, no_jitter
from .waits import constant, expo, fibo

__all__ = ["on_exception", "on_predicate", "Details", "expo", "fibo", "constant",
           "full_jitter", "equal_jitter", "no_jitter", "set_enabled", "is_enabled"]

Handlers = Union[None, Callable[[Details], Any], Sequence[Callable[[Details], Any]]]


def set_enabled(value: bool) -> None:
    """Globally switch retrying on or off (off: decorated functions are called once)."""
    _policy.enabled = bool(value)


def is_enabled() -> bool:
    return _policy.enabled


def _handlers(h: Handlers) -> Tuple[Callable[[Details], Any], ...]:
    if h is None:
        return ()
    return (h,) if callable(h) else tuple(h)


def _make(wait_gen, wait_gen_kwargs, max_tries, max_time, jitter, clock,
          on_success, on_backoff, on_giveup) -> _policy.Policy:
    if max_tries is not None and max_tries < 1:
        raise ValueError("max_tries must be >= 1")
    return _policy.Policy(wait_gen=wait_gen, max_tries=max_tries, max_time=max_time, jitter=jitter,
                          clock=clock, on_success=_handlers(on_success),
                          on_backoff=_handlers(on_backoff), on_giveup=_handlers(on_giveup))


def on_exception(
    wait_gen: Callable[..., Iterator[float]] = expo,
    exception: Union[Type[BaseException], Tuple[Type[BaseException], ...]] = Exception,
    *,
    max_tries: Optional[int] = None,
    max_time: Optional[float] = None,
    jitter: Optional[Callable[[float], float]] = full_jitter,
    giveup: Callable[[BaseException], bool] = lambda e: False,
    raise_on_giveup: bool = True,
    on_success: Handlers = None,
    on_backoff: Handlers = None,
    on_giveup: Handlers = None,
    sleep: Any = _decorate._UNSET,
    clock: Callable[[], float] = time.monotonic,
    **wait_gen_kwargs: Any,
):
    """Retry the decorated function when it raises `exception`."""
    policy = _make(wait_gen, wait_gen_kwargs, max_tries, max_time, jitter, clock,
                   on_success, on_backoff, on_giveup)

    def classify(is_exc: bool, payload: Any) -> bool:
        return is_exc and isinstance(payload, exception) and not giveup(payload)

    def decorator(func):
        return _decorate.build(func, policy, wait_gen_kwargs, sleep, classify, raise_on_giveup)

    return decorator


def on_predicate(
    wait_gen: Callable[..., Iterator[float]] = expo,
    predicate: Callable[[Any], bool] = lambda value: not value,
    *,
    max_tries: Optional[int] = None,
    max_time: Optional[float] = None,
    jitter: Optional[Callable[[float], float]] = full_jitter,
    on_success: Handlers = None,
    on_backoff: Handlers = None,
    on_giveup: Handlers = None,
    sleep: Any = _decorate._UNSET,
    clock: Callable[[], float] = time.monotonic,
    **wait_gen_kwargs: Any,
):
    """Retry the decorated function while `predicate(return_value)` is true."""
    policy = _make(wait_gen, wait_gen_kwargs, max_tries, max_time, jitter, clock,
                   on_success, on_backoff, on_giveup)

    def classify(is_exc: bool, payload: Any) -> bool:
        return (not is_exc) and bool(predicate(payload))

    def decorator(func):
        return _decorate.build(func, policy, wait_gen_kwargs, sleep, classify, False)

    return decorator
