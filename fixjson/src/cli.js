import { readFileSync } from 'node:fs';
import { gunzipSync } from 'node:zlib';
import { loadDict } from './dict.js';
import { decode } from './decode.js';
import { encode } from './encode.js';
import { toYaml } from './yaml.js';
import { splitMessages, validateRaw, detectDelimiter } from './wire.js';

export const HELP = `fixjson - FIX protocol messages to JSON or YAML, and back

usage: fixjson [options] [file ...]      (reads stdin when no file; .gz is unpacked)

  -d, --dict <name|path>   data dictionary: bundled name (FIX44) or QuickFIX XML file
  -f, --format <fmt>       json (default), jsonl, yaml
  -e, --enums              show enum descriptions instead of raw codes
      --flat               one flat object instead of Header/Body/Trailer
      --no-checksum        leave CheckSum(10) out of the output
      --validate           check BodyLength and CheckSum, exit 1 on problems
  -r, --reverse            read JSON / JSON lines and write FIX messages
      --delim <str>        field delimiter for --reverse output (default SOH; "|" ok)
  -h, --help               show this help
  -v, --version            show version
`;

export function parseArgs(argv) {
  const o = { files: [], format: 'json', delim: '\x01' };
  const need = (i, n) => {
    if (i + 1 >= argv.length) throw new Error(`${n} needs a value`);
    return argv[i + 1];
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case '-h': case '--help': o.help = true; break;
      case '-v': case '--version': o.version = true; break;
      case '-d': case '--dict': o.dict = need(i, a); i++; break;
      case '-f': case '--format': o.format = need(i, a); i++; break;
      case '-e': case '--enums': o.enums = true; break;
      case '--flat': o.flat = true; break;
      case '--no-checksum': o.checksum = false; break;
      case '--validate': o.validate = true; break;
      case '-r': case '--reverse': o.reverse = true; break;
      case '--delim': o.delim = need(i, a); i++; break;
      default:
        if (a.startsWith('-') && a !== '-') throw new Error(`unknown option ${a}`);
        o.files.push(a);
    }
  }
  if (!['json', 'jsonl', 'yaml'].includes(o.format)) throw new Error(`unknown format ${o.format}`);
  return o;
}

function readInput(file) {
  let buf = file === '-' ? readFileSync(0) : readFileSync(file);
  if (buf[0] === 0x1f && buf[1] === 0x8b) buf = gunzipSync(buf);
  return buf.toString('latin1');
}

const render = (items, format) => {
  if (format === 'jsonl') return items.map((m) => JSON.stringify(m) + '\n').join('');
  if (format === 'yaml') return items.map(toYaml).join('');
  return items.map((m) => JSON.stringify(m, null, 2) + '\n').join('');
};

/** Runs the CLI; returns { code, out, err } so it is easy to test. */
export function run(argv, pkgVersion = '0.0.0') {
  let o;
  try { o = parseArgs(argv); } catch (e) { return { code: 2, out: '', err: `fixjson: ${e.message}\n` }; }
  if (o.help) return { code: 0, out: HELP, err: '' };
  if (o.version) return { code: 0, out: pkgVersion + '\n', err: '' };
  try {
    const dict = o.dict ? loadDict(o.dict) : undefined;
    const inputs = o.files.length ? o.files : ['-'];
    const text = inputs.map(readInput).join('\n');
    if (o.reverse) {
      const objs = text.trim().startsWith('[') ? JSON.parse(text)
        : text.split('\n').filter((l) => l.trim()).map((l) => JSON.parse(l));
      const flat = objs.length === 1 && !Array.isArray(objs[0]) ? objs : objs.flat();
      return { code: 0, out: flat.map((m) => encode(m, dict, o.delim) + '\n').join(''), err: '' };
    }
    let err = '', bad = false;
    const items = [];
    for (const raw of splitMessages(text)) {
      if (o.validate) {
        for (const p of validateRaw(raw, detectDelimiter(raw))) { err += `fixjson: ${p}\n`; bad = true; }
      }
      const problems = [];
      items.push(decode(raw, dict, { enums: o.enums, flat: o.flat, checksum: o.checksum, problems }));
      for (const p of problems) { err += `fixjson: ${p}\n`; bad = bad || !!o.validate; }
    }
    return { code: bad ? 1 : 0, out: render(items, o.format), err };
  } catch (e) {
    return { code: 1, out: '', err: `fixjson: ${e.message}\n` };
  }
}
