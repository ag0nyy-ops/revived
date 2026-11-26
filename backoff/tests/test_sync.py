import unittest

import backoff


class FakeClock:
    def __init__(self):
        self.now = 0.0
        self.sleeps = []

    def __call__(self):
        return self.now

    def sleep(self, s):
        self.sleeps.append(s)
        self.now += s


def flaky(fails, exc=ValueError):
    state = {"calls": 0}

    def fn():
        state["calls"] += 1
        if state["calls"] <= fails:
            raise exc("boom")
        return state["calls"]

    return fn, state


class OnExceptionTests(unittest.TestCase):
    def setUp(self):
        self.clock = FakeClock()

    def deco(self, **kw):
        kw.setdefault("jitter", None)
        return backoff.on_exception(backoff.expo, ValueError, sleep=self.clock.sleep, clock=self.clock, **kw)

    def test_retries_until_success(self):
        fn, st = flaky(2)
        self.assertEqual(self.deco()(fn)(), 3)
        self.assertEqual(self.clock.sleeps, [1, 2])

    def test_gives_up_and_reraises(self):
        fn, st = flaky(10)
        with self.assertRaises(ValueError):
            self.deco(max_tries=3)(fn)()
        self.assertEqual(st["calls"], 3)

    def test_raise_on_giveup_false_returns_none(self):
        fn, _ = flaky(10)
        self.assertIsNone(self.deco(max_tries=2, raise_on_giveup=False)(fn)())

    def test_other_exceptions_propagate_immediately(self):
        fn, st = flaky(5, exc=KeyError)
        with self.assertRaises(KeyError):
            self.deco()(fn)()
        self.assertEqual(st["calls"], 1)

    def test_giveup_predicate(self):
        fn, st = flaky(5)
        with self.assertRaises(ValueError):
            self.deco(giveup=lambda e: True)(fn)()
        self.assertEqual(st["calls"], 1)

    def test_max_time_never_sleeps_past_deadline(self):
        fn, st = flaky(100)
        with self.assertRaises(ValueError):
            self.deco(max_time=5)(fn)()
        self.assertEqual(self.clock.sleeps, [1, 2])  # next wait of 4 would pass 5s
        self.assertLessEqual(self.clock.now, 5)

    def test_max_time_counts_call_duration_on_first_try(self):
        clock = self.clock

        def slow():
            clock.now += 10
            raise ValueError

        with self.assertRaises(ValueError):
            self.deco(max_time=5)(slow)()
        self.assertEqual(clock.sleeps, [])

    def test_handlers_receive_details(self):
        seen = []
        fn, _ = flaky(1)
        self.deco(on_backoff=lambda d: seen.append(("b", d.tries, d.wait, type(d.exception))),
                  on_success=lambda d: seen.append(("s", d.tries, d.value)))(fn)()
        self.assertEqual(seen, [("b", 1, 1, ValueError), ("s", 2, 2)])

    def test_giveup_handler_runs(self):
        seen = []
        fn, _ = flaky(9)
        with self.assertRaises(ValueError):
            self.deco(max_tries=2, on_giveup=seen.append)(fn)()
        self.assertEqual(len(seen), 1)
        self.assertIsNone(seen[0].wait)

    def test_invalid_max_tries(self):
        with self.assertRaises(ValueError):
            backoff.on_exception(max_tries=0)

    def test_wraps_metadata(self):
        @self.deco()
        def named():
            """doc"""
        self.assertEqual(named.__name__, "named")
        self.assertEqual(named.__doc__, "doc")


class OnPredicateTests(unittest.TestCase):
    def test_retries_while_falsy_by_default(self):
        clock = FakeClock()
        vals = iter([None, 0, "ok"])
        f = backoff.on_predicate(backoff.constant, jitter=None, interval=2,
                                 sleep=clock.sleep, clock=clock)(lambda: next(vals))
        self.assertEqual(f(), "ok")
        self.assertEqual(clock.sleeps, [2, 2])

    def test_returns_last_value_on_giveup(self):
        clock = FakeClock()
        f = backoff.on_predicate(backoff.constant, lambda v: v < 10, max_tries=3, jitter=None,
                                 sleep=clock.sleep, clock=clock)(lambda: 1)
        self.assertEqual(f(), 1)
        self.assertEqual(len(clock.sleeps), 2)


class StaticAndGlobalTests(unittest.TestCase):
    def test_staticmethod_and_classmethod(self):
        clock = FakeClock()
        deco = backoff.on_predicate(backoff.constant, max_tries=3, jitter=None, sleep=clock.sleep, clock=clock)

        class K:
            n = 0

            @staticmethod
            @deco
            def s():
                K.n += 1
                return K.n >= 3

            @classmethod
            @deco
            def c(cls):
                return True

        self.assertTrue(K.s())
        self.assertTrue(K.c())
        self.assertTrue(K().s())

    def test_global_disable(self):
        fn, st = flaky(5)
        deco = backoff.on_exception(backoff.expo, ValueError, sleep=lambda s: None)
        backoff.set_enabled(False)
        try:
            self.assertFalse(backoff.is_enabled())
            with self.assertRaises(ValueError):
                deco(fn)()
            self.assertEqual(st["calls"], 1)
        finally:
            backoff.set_enabled(True)


if __name__ == "__main__":
    unittest.main()
