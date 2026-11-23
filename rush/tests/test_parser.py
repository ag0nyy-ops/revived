import unittest

from rush_cli.parser import RushfileError, parse

SAMPLE = """\
# comment
a: |
    echo one
    echo two

b: |
    @a
    echo three
#c: |
    echo off
d: echo inline
"""


class ParserTests(unittest.TestCase):
    def test_blocks_and_order(self):
        t = parse(SAMPLE)
        self.assertEqual(list(t), ["a", "b", "c", "d"])
        self.assertEqual(t["a"].commands, ["echo one", "echo two"])

    def test_deps_excluded_from_script(self):
        t = parse(SAMPLE)
        self.assertEqual(t["b"].deps, ["a"])
        self.assertEqual(t["b"].script, "echo three\n")

    def test_disabled(self):
        t = parse(SAMPLE)
        self.assertTrue(t["c"].disabled)
        self.assertFalse(t["a"].disabled)

    def test_inline_command(self):
        self.assertEqual(parse(SAMPLE)["d"].commands, ["echo inline"])

    def test_missing_marker_reports_line(self):
        with self.assertRaisesRegex(RushfileError, "line 1.*name: \\|"):
            parse("name:\n    echo hi\n")

    def test_duplicate(self):
        with self.assertRaisesRegex(RushfileError, "duplicate"):
            parse("a: |\n  x\na: |\n  y\n")

    def test_garbage(self):
        with self.assertRaisesRegex(RushfileError, "line 1"):
            parse("???\n")

    def test_unicode_text(self):
        self.assertEqual(parse("a: |\n    echo héllo ✓\n")["a"].commands, ["echo héllo ✓"])


if __name__ == "__main__":
    unittest.main()
