import { emptyDict } from './dict.js';
import { checksum, SOH } from './wire.js';

function tagFor(dict, key) {
  const f = dict.fieldsByName.get(key);
  if (f) return f.tag;
  if (/^\d+$/.test(key)) return Number(key);
  throw new Error(`unknown field name: ${key}`);
}

function rawValue(dict, tag, v) {
  const f = dict.fieldsByTag.get(tag);
  if (f && typeof v === 'string') {
    for (const [code, desc] of f.values) if (desc === v) return code; // reverse --enums
  }
  return String(v);
}

function flatten(dict, obj, out) {
  for (const [key, v] of Object.entries(obj)) {
    const tag = tagFor(dict, key);
    if (Array.isArray(v)) {
      out.push([tag, String(v.length)]);
      for (const entry of v) flatten(dict, entry, out);
    } else {
      out.push([tag, rawValue(dict, tag, v)]);
    }
  }
}

/**
 * Encode a decoded object (sectioned or flat) back to FIX. BodyLength(9) and
 * CheckSum(10) are always recomputed. `delim` defaults to SOH.
 */
export function encode(obj, dict = emptyDict(), delim = SOH) {
  const header = [], body = [];
  const sectioned = obj.Header || obj.Body || obj.Trailer;
  if (sectioned) {
    flatten(dict, obj.Header ?? {}, header);
    flatten(dict, obj.Body ?? {}, body);
    const tr = [];
    flatten(dict, obj.Trailer ?? {}, tr);
    body.push(...tr.filter(([t]) => t !== 10));
  } else {
    const all = [];
    flatten(dict, obj, all);
    for (const p of all) (dict.header.tags.has(p[0]) ? header : body).push(p);
  }
  const take = (t) => header.find(([x]) => x === t);
  const begin = take(8);
  if (!begin) throw new Error('BeginString(8) is required');
  const rest = [...header.filter(([t]) => t !== 8 && t !== 9 && t !== 10), ...body.filter(([t]) => t !== 10)];
  const bodyStr = rest.map(([t, v]) => `${t}=${v}${delim}`).join('');
  const head = `8=${begin[1]}${delim}9=${bodyStr.length}${delim}`;
  const msg = head + bodyStr;
  return `${msg}10=${checksum(msg.split(delim).join(SOH))}${delim}`;
}
