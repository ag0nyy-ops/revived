import unittest

from csscss_cli.analyze import find_matches
from csscss_cli.parser import parse


class AnalyzeTests(unittest.TestCase):
    def test_basic_match(self):
        rs = parse(".a{color:red;top:0;left:0} .b{color:red;top:0;left:0;x:y} .c{color:blue}")
        m = find_matches(rs, 3)
        self.assertEqual(len(m), 1)
        self.assertEqual(m[0].count, 3)

    def test_minimum(self):
        rs = parse(".a{color:red;top:0} .b{color:red;top:0}")
        self.assertEqual(len(find_matches(rs, 3)), 0)
        self.assertEqual(len(find_matches(rs, 2)), 1)

    def test_media_scoped(self):
        rs = parse("@media a{.a{color:red;top:0}} @media b{.b{color:red;top:0}}")
        self.assertEqual(find_matches(rs, 1), [])
        rs = parse("@media a{.a{color:red;top:0}} @media a{.b{color:red;top:0}}")
        self.assertEqual(len(find_matches(rs, 2)), 1)

    def test_same_selector_reported(self):
        rs = parse(".a{color:red;top:0} .a{color:red;top:0}")
        m = find_matches(rs, 2)
        self.assertEqual(len(m), 1)
        self.assertTrue(m[0].same_selector)

    def test_shorthand_equivalence(self):
        rs = parse(".a{margin:0 auto} .b{margin-left:auto;margin-right:auto;margin-top:0;margin-bottom:0}")
        self.assertEqual(len(find_matches(rs, 1)), 1)
        self.assertEqual(find_matches(rs, 1, expand=False), [])

    def test_important_differs(self):
        rs = parse(".a{color:red!important} .b{color:red}")
        self.assertEqual(find_matches(rs, 1), [])

    def test_ignore_glob(self):
        rs = parse(".a{color:red} .keep-x{color:red}")
        self.assertEqual(find_matches(rs, 1, ignore=[".keep-*"]), [])

    def test_sorted_by_count(self):
        rs = parse(".a{a:1;b:2} .b{a:1;b:2} .c{a:1;b:2;c:3} .d{a:1;b:2;c:3}")
        counts = [m.count for m in find_matches(rs, 2)]
        self.assertEqual(counts, sorted(counts, reverse=True))


if __name__ == "__main__":
    unittest.main()
