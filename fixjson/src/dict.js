import { readFileSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { parseXml } from './xml.js';

const DICT_DIR = join(dirname(fileURLToPath(import.meta.url)), '..', 'dict');

export const NUMERIC = new Set([
  'INT', 'LENGTH', 'SEQNUM', 'NUMINGROUP', 'DAYOFMONTH', 'FLOAT', 'PRICE', 'QTY',
  'AMT', 'PERCENTAGE', 'PRICEOFFSET', 'QUANTITY',
]);

// Used when no dictionary is given, so sections still split sensibly.
export const DEFAULT_HEADER = [8, 9, 35, 49, 56, 34, 52, 43, 97, 115, 128, 116, 129, 50, 57, 122, 142, 143, 144];
export const DEFAULT_TRAILER = [93, 89, 10];

export function emptyDict() {
  return {
    name: null,
    fieldsByTag: new Map(),
    fieldsByName: new Map(),
    header: { tags: new Set(DEFAULT_HEADER), groups: new Map() },
    trailer: { tags: new Set(DEFAULT_TRAILER), groups: new Map() },
    messages: new Map(),
  };
}

/** Resolve "FIX44" (bundled) or a path to a dictionary XML file. */
export function resolveDictPath(nameOrPath) {
  if (existsSync(nameOrPath)) return nameOrPath;
  const bundled = join(DICT_DIR, `${nameOrPath}.xml`);
  if (existsSync(bundled)) return bundled;
  throw new Error(`dictionary not found: ${nameOrPath}`);
}

export function loadDict(nameOrPath) {
  const path = resolveDictPath(nameOrPath);
  return parseDict(readFileSync(path, 'utf8'), nameOrPath);
}

export function parseDict(xml, name = null) {
  const root = parseXml(xml);
  if (!root || root.name !== 'fix') throw new Error('dictionary root must be <fix>');
  const section = (n) => root.children.find((c) => c.name === n);
  const d = emptyDict();
  d.name = name;

  for (const f of section('fields')?.children ?? []) {
    const tag = Number(f.attrs.number);
    const field = { tag, name: f.attrs.name, type: (f.attrs.type || 'STRING').toUpperCase(), values: new Map() };
    for (const v of f.children) if (v.name === 'value') field.values.set(v.attrs.enum, v.attrs.description);
    d.fieldsByTag.set(tag, field);
    d.fieldsByName.set(field.name, field);
  }

  const components = new Map();
  for (const c of section('components')?.children ?? []) components.set(c.attrs.name, c);

  const tagOf = (n) => {
    const f = d.fieldsByName.get(n);
    if (!f) throw new Error(`dictionary: unknown field ${n}`);
    return f.tag;
  };

  // Fill a container (tags + groups) from child elements, flattening components.
  const fill = (container, children, seen = new Set()) => {
    for (const el of children) {
      if (el.name === 'field') {
        const t = tagOf(el.attrs.name);
        container.tags.add(t);
        container.order.push(t);
      } else if (el.name === 'group') {
        const t = tagOf(el.attrs.name);
        const inner = { tags: new Set(), order: [], groups: new Map() };
        fill(inner, el.children, seen);
        container.tags.add(t);
        container.order.push(t);
        container.groups.set(t, inner);
      } else if (el.name === 'component') {
        const cn = el.attrs.name;
        const comp = components.get(cn);
        if (!comp) throw new Error(`dictionary: unknown component ${cn}`);
        if (seen.has(cn)) throw new Error(`dictionary: recursive component ${cn}`);
        fill(container, comp.children, new Set([...seen, cn]));
      }
    }
  };
  const make = (children) => {
    const c = { tags: new Set(), order: [], groups: new Map() };
    fill(c, children);
    return c;
  };

  const hdr = section('header');
  if (hdr) d.header = make(hdr.children);
  const trl = section('trailer');
  if (trl) d.trailer = make(trl.children);
  for (const m of section('messages')?.children ?? []) {
    const c = make(m.children);
    c.name = m.attrs.name;
    c.msgtype = m.attrs.msgtype;
    d.messages.set(m.attrs.msgtype, c);
  }
  return d;
}
