import io, sys, tempfile, unittest
from contextlib import redirect_stdout, redirect_stderr
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))
sys.path.insert(0, str(Path(__file__).parent))
from helpers import make
from eml2html.cli import main


def run(*argv):
    out, err = io.StringIO(), io.StringIO()
    with redirect_stdout(out), redirect_stderr(err):
        code = main(list(argv))
    return code, out.getvalue(), err.getvalue()


class CliTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name)
        self.eml = self.dir / "a.eml"
        self.eml.write_bytes(bytes(make(html="<p>hi</p>")))

    def tearDown(self):
        self.tmp.cleanup()

    def test_writes_next_to_input(self):
        code, _, _ = run(str(self.eml))
        self.assertEqual(code, 0)
        self.assertIn("<p>hi</p>", (self.dir / "a.html").read_text("utf-8"))

    def test_refuses_overwrite_without_force(self):
        run(str(self.eml))
        code, _, err = run(str(self.eml))
        self.assertEqual(code, 1)
        self.assertIn("--force", err)
        self.assertEqual(run(str(self.eml), "--force")[0], 0)

    def test_missing_and_wrong_extension_skipped(self):
        txt = self.dir / "b.txt"
        txt.write_text("x")
        code, _, err = run(str(self.dir / "nope.eml"), str(txt))
        self.assertEqual(code, 1)
        self.assertIn("does not exist", err)
        self.assertIn("not an .eml", err)
        self.assertFalse((self.dir / "b.html").exists())

    def test_out_dir_and_directory_input(self):
        out = self.dir / "out"
        code, _, _ = run(str(self.dir), "-o", str(out))
        self.assertEqual(code, 0)
        self.assertTrue((out / "a.html").exists())

    def test_recursive(self):
        sub = self.dir / "sub"
        sub.mkdir()
        (sub / "c.eml").write_bytes(self.eml.read_bytes())
        run(str(self.dir), "-r")
        self.assertTrue((sub / "c.html").exists())

    def test_stdout(self):
        code, out, _ = run(str(self.eml), "--stdout")
        self.assertEqual(code, 0)
        self.assertIn("<p>hi</p>", out)
        self.assertFalse((self.dir / "a.html").exists())


if __name__ == "__main__":
    unittest.main()
