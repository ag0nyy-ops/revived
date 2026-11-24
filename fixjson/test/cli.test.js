import test from 'node:test';
import assert from 'node:assert/strict';
import { writeFileSync, mkdtempSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { run, parseArgs } from '../src/cli.js';
import { toYaml } from '../src/yaml.js';
import { sample } from './helpers.js';

const dir = mkdtempSync(join(tmpdir(), 'fixjson-'));
const file = join(dir, 'm.fix');
writeFileSync(file, sample + '\n' + sample + '\n');

test('json output for two messages', () => {
  const r = run(['-d', 'FIX44', '-f', 'jsonl', file]);
  const lines = r.out.trim().split('\n').map((l) => JSON.parse(l));
  assert.equal(lines.length, 2);
  assert.equal(lines[0].Body.NoMDEntries.length, 2);
});

test('gzip input is unpacked', () => {
  const gz = join(dir, 'm.gz');
  writeFileSync(gz, gzipSync(sample));
  assert.equal(JSON.parse(run(['--flat', gz]).out)['49'], 'A');
});

test('--validate exits 1 on a bad checksum', () => {
  const r = run(['--validate', file]);
  assert.equal(r.code, 1);
  assert.match(r.err, /CheckSum|BodyLength/);
});

test('--reverse produces valid FIX', () => {
  const json = join(dir, 'm.json');
  writeFileSync(json, run(['-d', 'FIX44', '-f', 'jsonl', file]).out);
  const r = run(['-d', 'FIX44', '-r', '--delim', '|', json]);
  assert.equal(r.code, 0);
  const back = join(dir, 'back.fix');
  writeFileSync(back, r.out);
  assert.equal(run(['--validate', back]).code, 0);
});

test('usage errors', () => {
  assert.equal(run(['--bogus']).code, 2);
  assert.equal(run(['-f', 'xml']).code, 2);
  assert.equal(run(['-d']).code, 2);
  assert.match(run(['-h']).out, /usage/);
  assert.equal(parseArgs(['a', 'b']).files.length, 2);
});

test('yaml quotes ambiguous strings and nests arrays', () => {
  const y = toYaml({ a: '123', b: 'yes', c: [{ x: 1, y: 'z' }], d: [] });
  assert.equal(y, '---\na: "123"\nb: "yes"\nc:\n  - x: 1\n    y: z\nd: []\n');
});
