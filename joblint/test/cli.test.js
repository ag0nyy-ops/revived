import test from 'node:test';
import assert from 'node:assert/strict';
import { writeFileSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { run } from '../src/cli.js';

const sink = () => ({ buf: '', write(s) { this.buf += s; } });
function cli(args, stdin = '') {
  const stdout = sink();
  const stderr = sink();
  const code = run(args, { stdout, stderr, readInput: () => stdin });
  return { code, out: stdout.buf, err: stderr.buf };
}
const dir = mkdtempSync(join(tmpdir(), 'joblint-'));
const file = (name, body) => { const p = join(dir, name); writeFileSync(p, body); return p; };

test('clean stdin exits 0', () => {
  const r = cli([], 'Backend engineer, 80k-95k');
  assert.equal(r.code, 0);
  assert.match(r.out, /No issues found/);
});

test('error finding exits 1 by default', () => {
  const r = cli([], 'Salesman wanted');
  assert.equal(r.code, 1);
  assert.match(r.out, /<stdin>:1:1\s+error/);
});

test('warnings alone do not fail by default but do with --fail-on warning', () => {
  assert.equal(cli([], 'ninja').code, 0);
  assert.equal(cli(['--fail-on', 'warning'], 'ninja').code, 1);
  assert.equal(cli(['--fail-on', 'none'], 'salesman').code, 0);
});

test('lints several files and names them in output', () => {
  const a = file('a.txt', 'a ninja');
  const b = file('b.txt', 'hello');
  const r = cli([a, b]);
  assert.match(r.out, /a\.txt:1:3/);
  assert.doesNotMatch(r.out, /b\.txt:/);
});

test('json format is parseable', () => {
  const r = cli(['-f', 'json'], 'urgent');
  const data = JSON.parse(r.out);
  assert.equal(data.files[0].file, '<stdin>');
  assert.equal(data.files[0].findings[0].rule, 'urgent');
  assert.equal(data.files[0].summary.warning, 1);
});

test('config can turn rules off, change levels, and add custom rules', () => {
  const cfg = file('c.json', JSON.stringify({
    rules: { 'hype-title': 'off', urgent: 'error' },
    custom: [{ id: 'no-php', level: 'error', message: 'No PHP', terms: ['php'] }],
  }));
  const r = cli(['-c', cfg, '-f', 'json'], 'ninja urgent PHP');
  const found = JSON.parse(r.out).files[0].findings.map((f) => `${f.rule}:${f.level}`);
  assert.deepEqual(found, ['urgent:error', 'no-php:error']);
});

test('bad config and bad input give exit 2', () => {
  const bad = file('bad.json', JSON.stringify({ rules: { nope: 'off' } }));
  assert.equal(cli(['-c', bad]).code, 2);
  assert.equal(cli([join(dir, 'missing.txt')]).code, 2);
  assert.equal(cli(['--format', 'xml']).code, 2);
  assert.equal(cli(['--bogus']).code, 2);
});

test('--list-rules and --help exit 0', () => {
  assert.match(cli(['--list-rules']).out, /hype-title/);
  assert.match(cli(['--help']).out, /Usage: joblint/);
});
