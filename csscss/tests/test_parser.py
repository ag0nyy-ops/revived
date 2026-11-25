import unittest

from csscss_cli.parser import CssError, parse, strip_comments


class ParserTests(unittest.TestCase):
    def test_comments_do_not_leak_into_selectors(self):
        rs = parse("/* hi { */ .a { color: red } /* x */ .b { color: blue }")
        self.assertEqual([r.label for r in rs], [".a", ".b"])

    def test_charset_then_comment_with_code(self):
        css = '@charset "utf-8";\n/* .x { color: red } */\n.a { color: red }'
        rs = parse(css)
        self.assertEqual(len(rs), 1)
        self.assertEqual(rs[0].line, 3)

    def test_selector_list_and_lines(self):
        rs = parse("a,\nb { color: red }\n\n.c { margin: 0 }")
        self.assertEqual(rs[0].selectors, ["a", "b"])
        self.assertEqual([r.line for r in rs], [1, 4])

    def test_media_context(self):
        rs = parse("@media (min-width: 10px) { .a { color: red } } .b { color: red }")
        self.assertEqual(rs[0].context, ("@media (min-width: 10px)",))
        self.assertEqual(rs[1].context, ())

    def test_nested_at_rules(self):
        rs = parse("@media a { @supports (x: y) { .a { top: 0 } } }")
        self.assertEqual(len(rs[0].context), 2)

    def test_font_face_and_keyframes_skipped(self):
        rs = parse("@font-face { font-family: x } @keyframes k { from { top: 0 } to { top: 1px } } .a { top: 0 }")
        self.assertEqual([r.label for r in rs], [".a"])

    def test_semicolon_in_url_and_string(self):
        rs = parse('.a { background: url(data:image/png;base64,AAA); content: ";" }')
        d = {x.prop: x.value for x in rs[0].declarations}
        self.assertEqual(d["background"], "url(data:image/png;base64,AAA)")
        self.assertEqual(d["content"], '";"')

    def test_important_and_case(self):
        d = parse(".a { COLOR: Red ! important }")[0].declarations[0]
        self.assertEqual((d.prop, d.value, d.important), ("color", "Red", True))

    def test_missing_final_semicolon_and_whitespace(self):
        d = parse(".a { color:   red;\n margin : 0 }")[0].declarations
        self.assertEqual([x.key() for x in d], ["color:red", "margin:0"])

    def test_unterminated_block(self):
        with self.assertRaises(CssError):
            parse(".a { color: red")

    def test_stray_brace(self):
        with self.assertRaises(CssError):
            parse(".a { color: red } }")

    def test_strip_comments_keeps_newlines(self):
        self.assertEqual(strip_comments("a/*\n\n*/b").count("\n"), 2)


if __name__ == "__main__":
    unittest.main()
