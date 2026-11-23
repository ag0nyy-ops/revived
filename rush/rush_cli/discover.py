"""Locate and read the rushfile."""
from __future__ import annotations

import os
from typing import Optional, Tuple

NAMES = ("rushfile.yml", "rushfile.yaml")


def find(start: str) -> Optional[str]:
    """Walk up from start looking for a rushfile."""
    cur = os.path.abspath(start)
    while True:
        for n in NAMES:
            p = os.path.join(cur, n)
            if os.path.isfile(p):
                return p
        parent = os.path.dirname(cur)
        if parent == cur:
            return None
        cur = parent


def read(path: str) -> str:
    with open(path, encoding="utf-8") as fh:
        return fh.read()
