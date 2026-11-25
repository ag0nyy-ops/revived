import { summarise } from './lint.js';

const COLORS = { error: '\x1b[31m', warning: '\x1b[33m', notice: '\x1b[36m', dim: '\x1b[2m', reset: '\x1b[0m' };

export function text(results, { color = false } = {}) {
  const c = (k, s) => (color ? `${COLORS[k]}${s}${COLORS.reset}` : s);
  const out = [];
  let total = { error: 0, warning: 0, notice: 0 };
  for (const { file, findings } of results) {
    for (const f of findings) {
      out.push(`${file}:${f.line}:${f.column}  ${c(f.level, f.level.padEnd(7))} ${f.message} ${c('dim', `(${f.rule}: "${f.match}")`)}`);
    }
    const s = summarise(findings);
    for (const k of Object.keys(total)) total[k] += s[k];
  }
  const n = total.error + total.warning + total.notice;
  out.push(n === 0
    ? 'No issues found.'
    : `${n} issue${n === 1 ? '' : 's'}: ${total.error} error, ${total.warning} warning, ${total.notice} notice`);
  return out.join('\n') + '\n';
}

export function json(results, { pretty = false } = {}) {
  const files = results.map(({ file, findings }) => ({ file, summary: summarise(findings), findings }));
  return JSON.stringify({ files }, null, pretty ? 2 : 0) + '\n';
}
