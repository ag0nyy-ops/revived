import test from 'node:test';
import assert from 'node:assert/strict';
import { compile, lint, summarise } from '../src/lint.js';
import { rules } from '../src/rules.js';

const compiled = rules.map(compile);
const run = (text) => lint(text, compiled);
const ids = (text) => run(text).map((f) => f.rule);

test('matches whole words only', () => {
  assert.deepEqual(ids('We need a Broadcaster and a bromide expert'), []);
  assert.deepEqual(ids('hey bro'), ['gendered-address']);
});

test('is case-insensitive and keeps original text', () => {
  const [f] = run('We want a ROCKSTAR developer');
  assert.equal(f.rule, 'hype-title');
  assert.equal(f.match, 'ROCKSTAR');
});

test('matches multi-word phrases across extra whitespace', () => {
  assert.deepEqual(ids('work   hard play hard'), ['overwork']);
});

test('curly quotes and en dashes still match', () => {
  assert.deepEqual(ids('We’re a family here'), ['family-culture']);
  assert.deepEqual(ids('a fast–paced team'), ['fast-paced']);
});

test('reports line and column', () => {
  const [f] = run('Great team\nPlease apply\n  Looking for a ninja');
  assert.equal(f.line, 3);
  assert.equal(f.column, 17);
});

test('flags 10+ years but not 5 years', () => {
  assert.deepEqual(ids('10+ years of Go'), ['years-experience']);
  assert.deepEqual(ids('15 years of React'), ['years-experience']);
  assert.deepEqual(ids('5 years of Go'), []);
});

test('symbol-only terms are matched', () => {
  assert.deepEqual(ids('on call 24/7'), ['overwork']);
});

test('findings are ordered by position', () => {
  const r = run('urgent ninja needed asap');
  assert.deepEqual(r.map((f) => f.match), ['urgent', 'ninja', 'asap']);
});

test('summarise counts levels', () => {
  assert.deepEqual(summarise(run('urgent salesman, competitive salary')), { error: 1, warning: 1, notice: 2 });
});

test('clean text has no findings', () => {
  assert.deepEqual(run('Backend engineer. Salary 70-90k. Remote friendly.'), []);
});

test('compile rejects bad rules', () => {
  assert.throws(() => compile({ id: 'x', level: 'bad', terms: ['a'] }), /invalid level/);
  assert.throws(() => compile({ id: 'x', level: 'error' }), /needs/);
});
