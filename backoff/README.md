# backoff

A typed, dependency-free rebuild of [litl/backoff](https://github.com/litl/backoff), the Python retry-with-backoff decorator library.

- **Original:** litl/backoff, MIT license, © its authors. This is a from-scratch rewrite; no original source was copied.
- **Why it died:** the repository was archived in August 2025 after the last release in 2022-2024 and an open "maintenance help?" issue went unanswered. Dozens of issues (type hints, `max_time` semantics, wall-clock timing, staticmethods, disabling in tests) were never resolved.

## What's improved

- `max_time` is a hard deadline: a wait that would pass it is never started, and call duration counts from the first try (original #186).
- Monotonic clock by default, injectable via `clock=` (original #149).
- `sleep=` is injectable (sync or async) and `backoff.set_enabled(False)` turns all retrying off globally, so tests don't wait (original #125, #206).
- Works on `staticmethod`, `classmethod`, plain and `async def` functions (original #200).
- Handlers receive a `Details` dataclass: target, args, kwargs, tries, elapsed, wait, value, exception (original #208, #221).
- Handlers may be coroutines in async code; `CancelledError` is never retried.
- Fully typed with `py.typed`, zero dependencies, Python 3.10+.

## Usage

```python
import backoff

@backoff.on_exception(backoff.expo, ConnectionError, max_tries=5, max_time=30)
def fetch(): ...

@backoff.on_predicate(backoff.fibo, lambda r: r.status == 503, max_tries=4, jitter=backoff.equal_jitter)
async def poll(): ...

@backoff.on_exception(backoff.constant, OSError, interval=2,
                      on_backoff=lambda d: print(f"try {d.tries} failed, waiting {d.wait:.1f}s"))
def save(): ...
```

Extra keyword arguments (such as `interval`, `base`, `factor`, `max_value`) go to the wait generator. `jitter=None` disables jitter. `on_exception` re-raises on give-up unless `raise_on_giveup=False`; `on_predicate` returns the last value.

## Tests

```
PYTHONPATH=src python3 -m unittest discover -s tests
```
