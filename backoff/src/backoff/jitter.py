"""Jitter strategies: functions mapping a computed wait to the wait actually used."""
from __future__ import annotations

import random


def no_jitter(value: float) -> float:
    return value


def full_jitter(value: float) -> float:
    """Uniform in [0, value]."""
    return random.uniform(0, value)


def equal_jitter(value: float) -> float:
    """Half fixed, half random: uniform in [value/2, value]."""
    return value / 2 + random.uniform(0, value / 2)
