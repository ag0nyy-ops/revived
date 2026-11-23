import unittest

from rush_cli.graph import order
from rush_cli.parser import RushfileError, parse


def tasks(text):
    return parse(text)


class GraphTests(unittest.TestCase):
    def test_deps_first_and_deduped(self):
        t = tasks("a: |\n  echo\nb: |\n  @a\n  echo\nc: |\n  @a\n  @b\n  echo\n")
        self.assertEqual(order(t, ["c"]), ["a", "b", "c"])

    def test_no_deps(self):
        t = tasks("a: |\n  echo\nb: |\n  @a\n  echo\n")
        self.assertEqual(order(t, ["b"], with_deps=False), ["b"])

    def test_cycle(self):
        t = tasks("a: |\n  @b\n  x\nb: |\n  @a\n  y\n")
        with self.assertRaisesRegex(RushfileError, "a -> b -> a"):
            order(t, ["a"])

    def test_unknown_names_requirer(self):
        t = tasks("a: |\n  @zzz\n  x\n")
        with self.assertRaisesRegex(RushfileError, "zzz.*needed by 'a'"):
            order(t, ["a"])

    def test_disabled_rejected(self):
        t = tasks("#a: |\n  x\n")
        with self.assertRaisesRegex(RushfileError, "disabled"):
            order(t, ["a"])


if __name__ == "__main__":
    unittest.main()
