import test from 'node:test';
import assert from 'node:assert/strict';
import { decode } from '../src/decode.js';
import { emptyDict } from '../src/dict.js';
import { dict, sample } from './helpers.js';

test('splits header, body and trailer', () => {
  const m = decode(sample, dict);
  assert.equal(m.Header.SenderCompID, 'A');
  assert.equal(m.Header.MsgSeqNum, 2);
  assert.equal(m.Body.MDReqID, 'r');
  assert.equal(m.Trailer.CheckSum, '000');
});

test('repeating groups become arrays of entries', () => {
  const g = decode(sample, dict).Body.NoMDEntries;
  assert.equal(g.length, 2);
  assert.deepEqual(g[1], { MDUpdateAction: '1', MDEntryType: '1', Symbol: 'EURUSD', MDEntryPx: 1.2, MDEntrySize: 7 });
});

test('fields after a group stay in the body', () => {
  const m = decode('8=FIX.4.4|35=W|268=1|269=0|270=1|55=X|58=after|10=1|', dict);
  assert.equal(m.Body.Text, 'after');
  assert.equal(m.Body.NoMDEntries.length, 1);
});

test('works without a dictionary using tag numbers', () => {
  const m = decode('8=FIX.4.4|35=0|49=A|9999=hello|10=5|');
  assert.equal(m.Header['49'], 'A');
  assert.equal(m.Body['9999'], 'hello');
  assert.equal(m.Trailer['10'], '5');
});

test('enums option, flat option, checksum removal', () => {
  const m = decode(sample, dict, { enums: true, flat: true, checksum: false });
  assert.equal(m.MsgType, 'MARKET_DATA_INCREMENTAL');
  assert.equal(m.CheckSum, undefined);
  assert.equal(m.NoMDEntries[0].MDEntryType, 'BID');
});

test('short group is reported as a problem', () => {
  const problems = [];
  decode('8=FIX.4.4|35=X|268=3|279=0|269=0|10=1|', dict, { problems });
  assert.match(problems[0], /declared 3, found 1/);
});

test('real SOH delimiter and caret form are detected', () => {
  assert.equal(decode(sample.replaceAll('|', '\x01'), dict).Header.SenderCompID, 'A');
  assert.equal(decode(sample.replaceAll('|', '^A'), dict).Header.SenderCompID, 'A');
});

test('malformed field throws', () => {
  assert.throws(() => decode('8=FIX.4.4|oops|', emptyDict()), /malformed/);
});
