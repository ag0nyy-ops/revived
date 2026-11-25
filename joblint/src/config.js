import { readFileSync } from 'node:fs';
import { rules as builtin, LEVELS } from './rules.js';
import { compile } from './lint.js';

// Config shape:
// {
//   "rules": { "<id>": "off" | "notice" | "warning" | "error" },
//   "custom": [{ "id", "level", "message", "terms": [...] | "pattern": "..." }]
// }
export function loadConfig(path) {
  let raw;
  try {
    raw = JSON.parse(readFileSync(path, 'utf8'));
  } catch (err) {
    throw new Error(`cannot read config ${path}: ${err.message}`);
  }
  return raw;
}

export function buildRules(config = {}) {
  const overrides = config.rules ?? {};
  const known = new Set(builtin.map((r) => r.id));
  const custom = config.custom ?? [];
  for (const c of custom) known.add(c.id);
  for (const [id, value] of Object.entries(overrides)) {
    if (!known.has(id)) throw new Error(`config: unknown rule "${id}"`);
    if (value !== 'off' && !LEVELS.includes(value)) {
      throw new Error(`config: rule "${id}" has invalid level "${value}"`);
    }
  }
  return [...builtin, ...custom]
    .filter((r) => overrides[r.id] !== 'off')
    .map((r) => compile({ ...r, level: overrides[r.id] ?? r.level }));
}
