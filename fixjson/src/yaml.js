// Small YAML emitter for the plain objects/arrays/scalars fixjson produces.
const PLAIN = /^[A-Za-z_][\w.\-\/]*$/;
const RESERVED = /^(true|false|null|yes|no|on|off|~)$/i;

function scalar(v) {
  if (typeof v === 'number' || typeof v === 'boolean') return String(v);
  if (v === null) return 'null';
  const s = String(v);
  return PLAIN.test(s) && !RESERVED.test(s) ? s : JSON.stringify(s);
}

function emit(v, indent) {
  const pad = '  '.repeat(indent);
  const lines = [];
  if (Array.isArray(v)) {
    for (const item of v) {
      if (item && typeof item === 'object') {
        const inner = emit(item, indent + 1).split('\n');
        lines.push(`${pad}- ${inner[0].trimStart()}`, ...inner.slice(1));
      } else lines.push(`${pad}- ${scalar(item)}`);
    }
  } else {
    for (const [k, val] of Object.entries(v)) {
      const key = PLAIN.test(k) ? k : JSON.stringify(k);
      if (val && typeof val === 'object') {
        if (Array.isArray(val) ? val.length === 0 : Object.keys(val).length === 0) {
          lines.push(`${pad}${key}: ${Array.isArray(val) ? '[]' : '{}'}`);
        } else lines.push(`${pad}${key}:`, emit(val, indent + 1));
      } else lines.push(`${pad}${key}: ${scalar(val)}`);
    }
  }
  return lines.join('\n');
}

export const toYaml = (obj) => `---\n${emit(obj, 0)}\n`;
