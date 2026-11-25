import test from 'node:test';
import assert from 'node:assert/strict';
import { rules, LEVELS } from '../src/rules.js';
import { compile, lint } from '../src/lint.js';

test('rule ids are unique and well formed', () => {
  const seen = new Set();
  for (const r of rules) {
    assert.ok(!seen.has(r.id), `duplicate ${r.id}`);
    seen.add(r.id);
    assert.ok(LEVELS.includes(r.level), r.id);
    assert.ok(r.message && r.category, r.id);
  }
});

test('every term in every rule matches itself', () => {
  for (const r of rules) {
    const c = compile(r);
    for (const t of r.terms ?? []) {
      const hits = lint(`We want ${t} here`, [c]);
      assert.ok(hits.length >= 1, `${r.id} does not match "${t}"`);
    }
  }
});
