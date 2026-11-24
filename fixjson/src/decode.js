import { NUMERIC, emptyDict } from './dict.js';
import { parsePairs, detectDelimiter } from './wire.js';

function convert(field, value, opts) {
  if (!field) return value;
  if (opts.enums && field.values.has(value)) return field.values.get(value);
  if (NUMERIC.has(field.type) && value !== '' && Number.isFinite(Number(value))) {
    // keep very long numbers (e.g. 23-digit timestamps typed wrongly) as text
    return /^-?\d+$/.test(value) && value.length > 15 ? value : Number(value);
  }
  return value;
}

const keyOf = (dict, tag) => dict.fieldsByTag.get(tag)?.name ?? String(tag);

/** Read one repeating group starting at the NumInGroup pair; returns [entries, nextIndex]. */
function readGroup(pairs, i, sub, dict, opts) {
  const n = Number(pairs[i][1]);
  const entries = [];
  i++;
  while (i < pairs.length && pairs[i][0] === sub.order[0] && entries.length < n) {
    const entry = {};
    i = readEntry(pairs, i, sub, dict, opts, entry);
    entries.push(entry);
  }
  if (entries.length !== n) {
    opts.problems?.push(`group ${keyOf(dict, pairs[i - 1]?.[0])}: declared ${n}, found ${entries.length}`);
  }
  return [entries, i];
}

/** Read one group entry: starts at the delimiter field, ends at the next one or a foreign tag. */
function readEntry(pairs, i, container, dict, opts, out) {
  const first = container.order[0];
  while (i < pairs.length) {
    const [tag, value] = pairs[i];
    if (tag === first && Object.keys(out).length > 0) break;
    if (!container.tags.has(tag)) break;
    const sub = container.groups.get(tag);
    if (sub) {
      const [entries, next] = readGroup(pairs, i, sub, dict, opts);
      out[keyOf(dict, tag)] = entries;
      i = next;
    } else {
      out[keyOf(dict, tag)] = convert(dict.fieldsByTag.get(tag), value, opts);
      i++;
    }
  }
  return i;
}

/**
 * Decode one raw message into { Header, Body, Trailer } (or flat with opts.flat).
 * opts: { enums, flat, checksum (keep tag 10, default true), problems }
 */
export function decode(raw, dict = emptyDict(), opts = {}) {
  const pairs = parsePairs(raw, opts.delim ?? detectDelimiter(raw));
  const msgType = pairs.find(([t]) => t === 35)?.[1];
  const bodyContainer = dict.messages.get(msgType) ?? { tags: new Set(), order: [], groups: new Map() };

  const header = {}, body = {}, trailer = {};
  let i = 0;
  while (i < pairs.length) {
    const tag = pairs[i][0];
    let target = body, container = bodyContainer;
    if (dict.trailer.tags.has(tag)) { target = trailer; container = dict.trailer; }
    else if (dict.header.tags.has(tag)) { target = header; container = dict.header; }
    const sub = container.groups.get(tag);
    if (sub) {
      const [entries, next] = readGroup(pairs, i, sub, dict, opts);
      target[keyOf(dict, tag)] = entries;
      i = next;
    } else {
      target[keyOf(dict, tag)] = convert(dict.fieldsByTag.get(tag), pairs[i][1], opts);
      i++;
    }
  }
  if (opts.checksum === false) delete trailer[keyOf(dict, 10)];
  if (opts.flat) return { ...header, ...body, ...trailer };
  const out = { Header: header, Body: body };
  if (Object.keys(trailer).length) out.Trailer = trailer;
  return out;
}
