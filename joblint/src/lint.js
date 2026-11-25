import { LEVELS } from './rules.js';

const escapeRe = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// Normalise typographic quotes and dashes so "we’re" matches "we're".
// Replacements are 1:1 in length so offsets stay valid.
export function normalise(text) {
  return text
    .replace(/[‘’]/g, "'")
    .replace(/[‐-―]/g, '-');
}

export function compile(rule) {
  if (!LEVELS.includes(rule.level)) {
    throw new Error(`rule ${rule.id}: invalid level "${rule.level}"`);
  }
  let source;
  if (rule.pattern) {
    source = rule.pattern;
  } else if (rule.terms?.length) {
    const alts = rule.terms
      .map((t) => escapeRe(normalise(t)).replace(/\s+/g, '\\s+'))
      .join('|');
    source = `(?<![\\w])(?:${alts})(?![\\w])`;
  } else {
    throw new Error(`rule ${rule.id}: needs "terms" or "pattern"`);
  }
  return { ...rule, regex: new RegExp(source, 'giu') };
}

export function lint(text, rules) {
  const input = normalise(text);
  const lineStarts = [0];
  for (let i = 0; i < input.length; i++) {
    if (input[i] === '\n') lineStarts.push(i + 1);
  }
  const findings = [];
  for (const rule of rules) {
    const re = new RegExp(rule.regex.source, rule.regex.flags);
    for (const m of input.matchAll(re)) {
      if (m[0] === '') continue;
      let lo = 0;
      let hi = lineStarts.length - 1;
      while (lo < hi) {
        const mid = (lo + hi + 1) >> 1;
        if (lineStarts[mid] <= m.index) lo = mid; else hi = mid - 1;
      }
      findings.push({
        rule: rule.id,
        level: rule.level,
        category: rule.category ?? 'custom',
        message: rule.message,
        match: text.slice(m.index, m.index + m[0].length),
        line: lo + 1,
        column: m.index - lineStarts[lo] + 1,
      });
    }
  }
  findings.sort((a, b) => a.line - b.line || a.column - b.column);
  return findings;
}

export function summarise(findings) {
  const counts = { error: 0, warning: 0, notice: 0 };
  for (const f of findings) counts[f.level]++;
  return counts;
}
