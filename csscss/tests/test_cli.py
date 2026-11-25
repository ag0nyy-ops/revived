import io
import json
import os
import tempfile
import unittest
from contextlib import redirect_stderr, redirect_stdout

from csscss_cli.cli import main

CSS = ".a{color:red;top:0;left:0}\n.b{color:red;top:0;left:0}\n"


def run(*args):
    out, err = io.StringIO(), io.StringIO()
    with redirect_stdout(out), redirect_stderr(err):
        code = main(list(args))
    return code, out.getvalue(), err.getvalue()


class CliTests(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.path = os.path.join(self.dir.name, "s.css")
        with open(self.path, "w") as f:
            f.write(CSS)

    def tearDown(self):
        self.dir.cleanup()

    def test_text_output_has_location(self):
        code, out, _ = run(self.path)
        self.assertEqual(code, 0)
        self.assertIn("s.css:1", out)
        self.assertIn("s.css:2", out)
        self.assertIn("share 3 rules", out)

    def test_verbose_lists_declarations(self):
        _, out, _ = run("-v", self.path)
        self.assertIn("color:red", out)

    def test_json(self):
        _, out, _ = run("--json", self.path)
        data = json.loads(out)
        self.assertEqual(data[0]["count"], 3)
        self.assertEqual(data[0]["b"]["line"], 2)

    def test_fail_on_duplicates(self):
        self.assertEqual(run("--fail-on-duplicates", self.path)[0], 1)
        self.assertEqual(run("--fail-on-duplicates", "-n", "9", self.path)[0], 0)

    def test_missing_file(self):
        code, _, err = run("/nonexistent/x.css")
        self.assertEqual(code, 2)
        self.assertIn("x.css", err)

    def test_parse_error_exit(self):
        bad = os.path.join(self.dir.name, "bad.css")
        with open(bad, "w") as f:
            f.write(".a { color: red")
        code, _, err = run(bad)
        self.assertEqual(code, 2)
        self.assertIn("unterminated", err)

    def test_multiple_files(self):
        other = os.path.join(self.dir.name, "o.css")
        with open(other, "w") as f:
            f.write(".c{color:red;top:0;left:0}")
        _, out, _ = run(self.path, other)
        self.assertIn("o.css:1", out)

    def test_bad_min(self):
        self.assertEqual(run("-n", "0", self.path)[0], 2)


if __name__ == "__main__":
    unittest.main()
