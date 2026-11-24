// Minimal XML reader: elements, attributes, self-closing tags. Enough for
// QuickFIX-style data dictionaries. Ignores comments, PIs and text nodes.
const ENT = { '&lt;': '<', '&gt;': '>', '&amp;': '&', '&quot;': '"', '&apos;': "'" };
const unescape = (s) => s.replace(/&(lt|gt|amp|quot|apos);/g, (m) => ENT[m]);

export function parseXml(text) {
  const root = { name: '#root', attrs: {}, children: [] };
  const stack = [root];
  const re = /<!--[\s\S]*?-->|<\?[\s\S]*?\?>|<!\[CDATA\[[\s\S]*?\]\]>|<!DOCTYPE[^>]*>|<(\/?)([A-Za-z_][\w.:-]*)((?:\s+[\w.:-]+\s*=\s*(?:"[^"]*"|'[^']*'))*)\s*(\/?)>/g;
  let m;
  while ((m = re.exec(text))) {
    if (!m[2]) continue;
    const [, closing, name, attrText, selfClose] = m;
    if (closing) {
      const top = stack.pop();
      if (!top || top.name !== name) throw new Error(`xml: unexpected </${name}>`);
      continue;
    }
    const attrs = {};
    for (const a of attrText.matchAll(/([\w.:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')/g)) {
      attrs[a[1]] = unescape(a[2] ?? a[3]);
    }
    const el = { name, attrs, children: [] };
    stack[stack.length - 1].children.push(el);
    if (!selfClose) stack.push(el);
  }
  if (stack.length !== 1) throw new Error(`xml: unclosed <${stack[stack.length - 1].name}>`);
  return root.children[0];
}
