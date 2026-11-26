import itertools
import unittest

import backoff


def take(gen, n):
    return list(itertools.islice(gen, n))


class WaitTests(unittest.TestCase):
    def test_expo(self):
        self.assertEqual(take(backoff.expo(), 5), [1, 2, 4, 8, 16])

    def test_expo_factor_base_cap(self):
        self.assertEqual(take(backoff.expo(base=3, factor=2, max_value=20), 5), [2, 6, 18, 20, 20])

    def test_fibo(self):
        self.assertEqual(take(backoff.fibo(), 6), [1, 1, 2, 3, 5, 8])
        self.assertEqual(take(backoff.fibo(max_value=4), 6), [1, 1, 2, 3, 4, 4])

    def test_constant(self):
        self.assertEqual(take(backoff.constant(0.5), 3), [0.5] * 3)

    def test_jitter_bounds(self):
        for _ in range(200):
            self.assertTrue(0 <= backoff.full_jitter(8) <= 8)
            self.assertTrue(4 <= backoff.equal_jitter(8) <= 8)
        self.assertEqual(backoff.no_jitter(3), 3)


if __name__ == "__main__":
    unittest.main()
