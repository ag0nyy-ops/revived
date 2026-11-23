import contextlib
import io
import os
import tempfile
import unittest

from rush_cli.cli import main

RUSHFILE = """\
ok: |
    echo ok-ran > out.txt
use: |
    @ok
    cp out.txt copy.txt
bad: |
    exit 3
after: |
    echo after > after.txt
#off: |
    echo no
uni: |
    echo héllo > uni.txt
"""


class CliTests(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.file = os.path.join(self.dir.name, "rushfile.yml")
        with open(self.file, "w", encoding="utf-8") as f:
            f.write(RUSHFILE)

    def run_cli(self, *args):
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            rc = main(["-f", self.file, *args])
        return rc, out.getvalue(), err.getvalue()

    def read(self, name):
        with open(os.path.join(self.dir.name, name), encoding="utf-8") as f:
            return f.read()

    def test_list_shows_disabled(self):
        rc, out, _ = self.run_cli("--list")
        self.assertEqual(rc, 0)
        self.assertIn("off (disabled)", out)

    def test_dry_run_orders_deps(self):
        rc, out, _ = self.run_cli("-n", "use")
        self.assertEqual(out.split(), ["ok", "use"])

    def test_runs_in_rushfile_dir_with_deps(self):
        rc, _, _ = self.run_cli("use")
        self.assertEqual(rc, 0)
        self.assertEqual(self.read("copy.txt"), "ok-ran\n")

    def test_failure_stops_and_propagates_status(self):
        rc, _, err = self.run_cli("bad", "after")
        self.assertEqual(rc, 3)
        self.assertIn("failed with status 3", err)
        self.assertFalse(os.path.exists(os.path.join(self.dir.name, "after.txt")))

    def test_ignore_errors_continues(self):
        rc, _, _ = self.run_cli("--ignore-errors", "bad", "after")
        self.assertEqual(rc, 3)
        self.assertEqual(self.read("after.txt"), "after\n")

    def test_unknown_task(self):
        rc, _, err = self.run_cli("nope")
        self.assertEqual(rc, 2)
        self.assertIn("unknown task 'nope'", err)

    def test_all_skips_disabled(self):
        rc, out, _ = self.run_cli("--all", "--dry-run")
        self.assertNotIn("off", out.split())

    def test_unicode_output_file(self):
        self.run_cli("uni")
        self.assertEqual(self.read("uni.txt"), "héllo\n")

    def test_path(self):
        _, out, _ = self.run_cli("--path")
        self.assertEqual(out.strip(), self.file)


if __name__ == "__main__":
    unittest.main()
