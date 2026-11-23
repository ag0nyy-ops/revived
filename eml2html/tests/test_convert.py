import sys, unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))
sys.path.insert(0, str(Path(__file__).parent))
from helpers import make
from eml2html.convert import convert, parse_bytes


class ConvertTests(unittest.TestCase):
    def test_html_preferred(self):
        m = make(plain="plain", html="<p>rich</p>")
        self.assertIn("<p>rich</p>", convert(m))

    def test_plain_fallback_escaped(self):
        out = convert(make(plain="a <b> & c"))
        self.assertIn("a &lt;b&gt; &amp; c", out)

    def test_latin1_charset(self):
        raw = (b"From: a@x\r\nSubject: s\r\nMIME-Version: 1.0\r\n"
               b"Content-Type: text/html; charset=iso-8859-1\r\n"
               b"Content-Transfer-Encoding: 8bit\r\n\r\n<p>caf\xe9</p>\r\n")
        self.assertIn("café", convert(parse_bytes(raw)))

    def test_unknown_charset_does_not_crash(self):
        raw = (b"From: a@x\r\nMIME-Version: 1.0\r\n"
               b"Content-Type: text/plain; charset=bogus-9\r\n"
               b"Content-Transfer-Encoding: 8bit\r\n\r\nhello\r\n")
        self.assertIn("hello", convert(parse_bytes(raw)))

    def test_cid_inlined(self):
        m = make(html='<img src="cid:logo1">')
        m.get_body().add_related(b"\x89PNG", "image", "png", cid="<logo1>")
        out = convert(m)
        self.assertIn("data:image/png;base64,", out)
        self.assertNotIn("cid:", out)

    def test_unknown_cid_left_alone(self):
        self.assertIn("cid:nope", convert(make(html='<img src="cid:nope">')))

    def test_headers(self):
        out = convert(make(plain="x", subject="A & B"), with_headers=True)
        self.assertIn("A &amp; B", out)
        self.assertIn("a@example.com", out)

    def test_headers_inserted_after_body_tag(self):
        out = convert(make(html="<html><body><p>x</p></body></html>"), True)
        self.assertTrue(out.index("<body>") < out.index("eml-headers"))

    def test_empty_message(self):
        self.assertNotIn("Traceback", convert(make()))


if __name__ == "__main__":
    unittest.main()
