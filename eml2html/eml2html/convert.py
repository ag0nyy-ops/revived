"""Turn a parsed email message into an HTML document."""
import base64
import html
import re
from email import policy
from email.message import EmailMessage


def _body_part(msg, kind):
    part = msg.get_body(preferencelist=(kind,))
    if part is None:
        return None
    try:
        return part.get_content()
    except (LookupError, UnicodeError):
        raw = part.get_payload(decode=True) or b""
        return raw.decode("utf-8", errors="replace")


def text_to_html(text):
    return "<pre style=\"white-space:pre-wrap\">" + html.escape(text) + "</pre>"


def body_html(msg):
    """Return an HTML body, preferring text/html, falling back to text/plain."""
    h = _body_part(msg, "html")
    if h is not None:
        return h
    t = _body_part(msg, "plain")
    if t is not None:
        return text_to_html(t)
    return ""


_CID = re.compile(r"""(["'(])cid:([^"')\s]+)""", re.I)


def inline_cids(body, msg):
    """Replace cid: references with data URIs built from related parts."""
    parts = {}
    for part in msg.walk():
        cid = part.get("Content-ID")
        if cid and not part.is_multipart():
            data = part.get_payload(decode=True)
            if data is not None:
                parts[cid.strip().strip("<>")] = (part.get_content_type(), data)

    def repl(m):
        hit = parts.get(m.group(2))
        if not hit:
            return m.group(0)
        ctype, data = hit
        return "%sdata:%s;base64,%s" % (
            m.group(1), ctype, base64.b64encode(data).decode("ascii"))

    return _CID.sub(repl, body)
