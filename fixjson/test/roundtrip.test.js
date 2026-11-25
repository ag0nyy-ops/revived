import test from 'node:test';
import assert from 'node:assert/strict';
import { decode } from '../src/decode.js';
import { encode } from '../src/encode.js';
import { checksum, validateRaw } from '../src/wire.js';
import { dict, sample } from './helpers.js';

const fixed = encode(decode(sample, dict), dict, '|');

test('encode recomputes a valid length and checksum', () => {
  assert.deepEqual(validateRaw(fixed, '|'), []);
  assert.notEqual(fixed, sample);
});

test('decode then encode is stable', () => {
  const again = encode(decode(fixed, dict), dict, '|');
  assert.equal(again, fixed);
});

test('round trip with enum names and flat objects', () => {
  const viaEnums = encode(decode(fixed, dict, { enums: true }), dict, '|');
  assert.equal(viaEnums, fixed);
  const viaFlat = encode(decode(fixed, dict, { flat: true }), dict, '|');
  assert.equal(viaFlat, fixed);
});

test('validateRaw flags tampering', () => {
  const bad = fixed.replace('EURUSD', 'EURUSX');
  assert.match(validateRaw(bad, '|').join(), /CheckSum/);
  assert.match(validateRaw(fixed.replace('9=', '9=1'), '|').join(), /BodyLength/);
});

test('checksum is byte sum mod 256', () => {
  assert.equal(checksum('A'), '065');
  assert.equal(checksum('A'.repeat(4)), '004');
});

test('unknown field names are rejected, BeginString required', () => {
  assert.throws(() => encode({ Body: { Nope: 1 } }, dict), /unknown field/);
  assert.throws(() => encode({ Body: { Symbol: 'X' } }, dict), /BeginString/);
});
