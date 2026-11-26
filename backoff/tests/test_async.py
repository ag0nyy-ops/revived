import asyncio
import unittest

import backoff


class AsyncTests(unittest.TestCase):
    def run_async(self, coro):
        return asyncio.run(coro)

    def test_retries_and_awaits_async_sleep_and_handlers(self):
        sleeps, events = [], []

        async def sleep(s):
            sleeps.append(s)

        async def on_backoff(d):
            events.append(d.tries)

        calls = {"n": 0}

        @backoff.on_exception(backoff.constant, OSError, interval=3, jitter=None,
                              sleep=sleep, on_backoff=on_backoff)
        async def op():
            calls["n"] += 1
            if calls["n"] < 3:
                raise OSError
            return "done"

        self.assertEqual(self.run_async(op()), "done")
        self.assertEqual(sleeps, [3, 3])
        self.assertEqual(events, [1, 2])

    def test_gives_up(self):
        async def sleep(s):
            pass

        @backoff.on_exception(backoff.expo, OSError, max_tries=2, sleep=sleep)
        async def op():
            raise OSError("x")

        with self.assertRaises(OSError):
            self.run_async(op())

    def test_cancellation_is_not_retried(self):
        calls = {"n": 0}

        @backoff.on_exception(backoff.constant, BaseException, interval=0, sleep=lambda s: None)
        async def op():
            calls["n"] += 1
            raise asyncio.CancelledError

        with self.assertRaises(asyncio.CancelledError):
            self.run_async(op())
        self.assertEqual(calls["n"], 1)

    def test_on_predicate_async(self):
        vals = iter([False, False, True])

        @backoff.on_predicate(backoff.constant, jitter=None, interval=0, sleep=lambda s: None)
        async def op():
            return next(vals)

        self.assertTrue(self.run_async(op()))


if __name__ == "__main__":
    unittest.main()
