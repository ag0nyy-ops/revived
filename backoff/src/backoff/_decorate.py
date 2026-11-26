"""Decorator factory shared by on_exception and on_predicate."""
from __future__ import annotations

import asyncio
import functools
import inspect
import time
from typing import Any, Callable, Optional

from . import _policy
from .details import Details

_UNSET = object()


def _sync_wrapper(func, policy, wait_kwargs, sleep, classify, raise_on_giveup):
    """classify(outcome_kind, payload) -> True when the outcome should be retried."""

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        if not _policy.enabled:
            return func(*args, **kwargs)
        gen = policy.wait_gen(**wait_kwargs)
        start = policy.clock()
        tries = 0
        while True:
            tries += 1
            value, exc = None, None
            try:
                value = func(*args, **kwargs)
            except BaseException as e:  # noqa: BLE001 - filtered by classify
                if not classify(True, e):
                    raise
                exc = e
            else:
                if not classify(False, value):
                    _policy.fire(policy.on_success, _details(func, args, kwargs, tries, policy, start, value=value))
                    return value
            elapsed = policy.clock() - start
            wait = policy.next_wait(gen, elapsed, tries)
            d = _details(func, args, kwargs, tries, policy, start, value=value, exception=exc, wait=wait)
            if wait is None:
                _policy.fire(policy.on_giveup, d)
                if exc is not None and raise_on_giveup:
                    raise exc
                return value
            _policy.fire(policy.on_backoff, d)
            sleep(wait)

    return wrapper


def _async_wrapper(func, policy, wait_kwargs, sleep, classify, raise_on_giveup):
    async def fire(handlers, details):
        for h in handlers:
            result = h(details)
            if inspect.isawaitable(result):
                await result

    @functools.wraps(func)
    async def wrapper(*args, **kwargs):
        if not _policy.enabled:
            return await func(*args, **kwargs)
        gen = policy.wait_gen(**wait_kwargs)
        start = policy.clock()
        tries = 0
        while True:
            tries += 1
            value, exc = None, None
            try:
                value = await func(*args, **kwargs)
            except asyncio.CancelledError:
                raise
            except BaseException as e:  # noqa: BLE001
                if not classify(True, e):
                    raise
                exc = e
            else:
                if not classify(False, value):
                    await fire(policy.on_success, _details(func, args, kwargs, tries, policy, start, value=value))
                    return value
            elapsed = policy.clock() - start
            wait = policy.next_wait(gen, elapsed, tries)
            d = _details(func, args, kwargs, tries, policy, start, value=value, exception=exc, wait=wait)
            if wait is None:
                await fire(policy.on_giveup, d)
                if exc is not None and raise_on_giveup:
                    raise exc
                return value
            await fire(policy.on_backoff, d)
            result = sleep(wait)
            if inspect.isawaitable(result):
                await result

    return wrapper


def _details(func, args, kwargs, tries, policy, start, **extra) -> Details:
    return Details(target=func, args=args, kwargs=kwargs, tries=tries,
                   elapsed=policy.clock() - start, **extra)


def build(func: Any, policy, wait_kwargs, sleep, classify, raise_on_giveup):
    """Wrap func (plain, async, staticmethod or classmethod)."""
    if isinstance(func, (staticmethod, classmethod)):
        return type(func)(build(func.__func__, policy, wait_kwargs, sleep, classify, raise_on_giveup))
    if inspect.iscoroutinefunction(func):
        return _async_wrapper(func, policy, wait_kwargs, sleep if sleep is not _UNSET else asyncio.sleep,
                              classify, raise_on_giveup)
    return _sync_wrapper(func, policy, wait_kwargs, time.sleep if sleep is _UNSET else sleep,
                         classify, raise_on_giveup)
