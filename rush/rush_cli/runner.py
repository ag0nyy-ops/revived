"""Execute task scripts through bash."""
from __future__ import annotations

import shutil
import subprocess
from typing import Optional

from .parser import Task


def shell() -> str:
    found = shutil.which("bash")
    if not found:
        raise RuntimeError("bash was not found on PATH")
    return found


def run_task(task: Task, cwd: Optional[str] = None, quiet: bool = False) -> int:
    """Run one task's script. Returns the exit status."""
    out = subprocess.DEVNULL if quiet else None
    proc = subprocess.run(
        [shell(), "-e", "-c", task.script],
        cwd=cwd, stdout=out, stderr=out, check=False,
    )
    return proc.returncode
