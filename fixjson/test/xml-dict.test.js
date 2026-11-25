import test from 'node:test';
import assert from 'node:assert/strict';
import { parseXml } from '../src/xml.js';
import { parseDict, loadDict } from '../src/dict.js';

test('xml: attributes, nesting, self-closing, comments', () => {
  const r = parseXml('<?xml version="1.0"?><!-- hi --><a x="1" y=\'2&amp;3\'><b/><c><d k="v"></d></c></a>');
  assert.equal(r.name, 'a');
  assert.equal(r.attrs.y, '2&3');
  assert.equal(r.children[1].children[0].attrs.k, 'v');
});

test('xml: mismatched close tag throws', () => {
  assert.throws(() => parseXml('<a><b></a>'));
});

test('dict: attribute order does not matter', () => {
  const d = parseDict(`<fix><fields><field type="INT" name="Foo" number="9001"/></fields>
    <messages><message msgtype="Z" name="Z"><field required="Y" name="Foo"/></message></messages></fix>`);
  assert.equal(d.fieldsByTag.get(9001).name, 'Foo');
  assert.ok(d.messages.get('Z').tags.has(9001));
});

test('dict: components are flattened, unknown ones rejected', () => {
  const d = loadDict('FIX44');
  assert.ok(d.messages.get('D').tags.has(55));
  assert.throws(() => parseDict('<fix><fields/><messages><message name="m" msgtype="m"><component name="Nope"/></message></messages></fix>'));
});

test('dict: bundled name and missing dictionary', () => {
  assert.ok(loadDict('FIX44').fieldsByName.has('Symbol'));
  assert.throws(() => loadDict('does-not-exist'), /not found/);
});
