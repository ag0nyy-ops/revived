const SOH = '\x01';

/** Guess the field delimiter from a raw message. */
export function detectDelimiter(text) {
  if (text.includes(SOH)) return SOH;
  if (text.includes('^A')) return '^A';
  if (/8=FIX[^|]*\|/.test(text)) return '|';
  return SOH;
}

/** Split a stream of text into raw message strings (each starts at 8=FIX). */
export function splitMessages(text) {
  const out = [];
  const re = /8=FIX/g;
  const starts = [];
  let m;
  while ((m = re.exec(text))) starts.push(m.index);
  starts.forEach((s, i) => {
    const raw = text.slice(s, starts[i + 1] ?? text.length).replace(/[\r\n]+$/, '');
    if (raw) out.push(raw);
  });
  return out;
}

/** Parse "tag=value<delim>..." into [[tag, value], ...]. */
export function parsePairs(raw, delim = detectDelimiter(raw)) {
  const pairs = [];
  for (const part of raw.split(delim)) {
    if (!part) continue;
    const eq = part.indexOf('=');
    if (eq < 1) throw new Error(`malformed field: ${JSON.stringify(part)}`);
    const tag = Number(part.slice(0, eq));
    if (!Number.isInteger(tag)) throw new Error(`non-numeric tag: ${JSON.stringify(part)}`);
    pairs.push([tag, part.slice(eq + 1)]);
  }
  return pairs;
}

export function checksum(str) {
  let sum = 0;
  for (let i = 0; i < str.length; i++) sum = (sum + (str.charCodeAt(i) & 0xff)) & 0xff;
  return String(sum).padStart(3, '0');
}

/** Check BodyLength(9) and CheckSum(10) of a raw message. Returns problems. */
export function validateRaw(raw, delim = detectDelimiter(raw)) {
  const problems = [];
  const sepLen = delim.length;
  const ckAt = raw.lastIndexOf(`${delim}10=`);
  const start9 = raw.indexOf(`${delim}9=`);
  if (ckAt < 0) return ['missing CheckSum(10)'];
  if (start9 < 0) return ['missing BodyLength(9)'];
  const lenEnd = raw.indexOf(delim, start9 + sepLen);
  const declared = Number(raw.slice(start9 + sepLen + 2, lenEnd));
  // body runs from after the delimiter ending field 9 up to and including the delimiter before 10=
  const body = raw.slice(lenEnd + sepLen, ckAt + sepLen);
  if (body.length !== declared) problems.push(`BodyLength is ${declared}, actual ${body.length}`);
  const want = checksum(raw.slice(0, ckAt + sepLen).split(delim).join(SOH));
  const got = raw.slice(ckAt + sepLen + 3, ckAt + sepLen + 6);
  if (want !== got) problems.push(`CheckSum is ${got}, expected ${want}`);
  return problems;
}

export { SOH };
