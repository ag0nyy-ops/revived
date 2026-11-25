import { readFileSync } from 'node:fs';
import { parseArgs } from 'node:util';
import { lint } from './lint.js';
import { buildRules, loadConfig } from './config.js';
import { rules as builtin, LEVELS } from './rules.js';
import * as report from './report.js';

const HELP = `Usage: joblint [options] [file...]

Lint job posts. Reads stdin when no file is given or when a file is "-".

Options:
  -c, --config <file>   JSON config (rule levels, custom rules)
  -f, --format <fmt>    text (default) or json
  -F, --fail-on <lvl>   exit 1 at this level or above: error (default),
                        warning, notice, none
      --pretty          indent JSON output
      --no-color        disable colour
      --list-rules      print the rule set and exit
  -h, --help            show this help
  -v, --version         show version

Exit codes: 0 ok, 1 findings at or above --fail-on, 2 usage or input error.
`;

const readStdin = () => readFileSync(0, 'utf8');

export function run(argv, { stdout = process.stdout, stderr = process.stderr, readInput = readStdin, isTTY = false } = {}) {
  let values, positionals;
  try {
    ({ values, positionals } = parseArgs({
      args: argv,
      allowPositionals: true,
      options: {
        config: { type: 'string', short: 'c' },
        format: { type: 'string', short: 'f', default: 'text' },
        'fail-on': { type: 'string', short: 'F', default: 'error' },
        pretty: { type: 'boolean' },
        color: { type: 'boolean', default: true },
        'no-color': { type: 'boolean' },
        'list-rules': { type: 'boolean' },
        help: { type: 'boolean', short: 'h' },
        version: { type: 'boolean', short: 'v' },
      },
    }));
  } catch (err) {
    stderr.write(`joblint: ${err.message}\n`);
    return 2;
  }
  if (values.help) { stdout.write(HELP); return 0; }
  if (values.version) {
    const pkg = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'));
    stdout.write(`${pkg.version}\n`);
    return 0;
  }
  if (values['list-rules']) {
    for (const r of builtin) stdout.write(`${r.id.padEnd(18)} ${r.level.padEnd(8)} ${r.category.padEnd(13)} ${r.message}\n`);
    return 0;
  }
  if (!['text', 'json'].includes(values.format)) {
    stderr.write(`joblint: unknown format "${values.format}"\n`);
    return 2;
  }
  const failOn = values['fail-on'];
  if (failOn !== 'none' && !LEVELS.includes(failOn)) {
    stderr.write(`joblint: invalid --fail-on "${failOn}"\n`);
    return 2;
  }

  let rules;
  try {
    rules = buildRules(values.config ? loadConfig(values.config) : {});
  } catch (err) {
    stderr.write(`joblint: ${err.message}\n`);
    return 2;
  }

  const inputs = positionals.length ? positionals : ['-'];
  const results = [];
  for (const name of inputs) {
    let content;
    try {
      content = name === '-' ? readInput() : readFileSync(name, 'utf8');
    } catch (err) {
      stderr.write(`joblint: cannot read ${name}: ${err.code ?? err.message}\n`);
      return 2;
    }
    results.push({ file: name === '-' ? '<stdin>' : name, findings: lint(content, rules) });
  }

  const color = values.color && !values['no-color'] && isTTY && !process.env.NO_COLOR;
  stdout.write(values.format === 'json'
    ? report.json(results, { pretty: values.pretty })
    : report.text(results, { color }));

  if (failOn === 'none') return 0;
  const threshold = LEVELS.indexOf(failOn);
  const failed = results.some((r) => r.findings.some((f) => LEVELS.indexOf(f.level) >= threshold));
  return failed ? 1 : 0;
}
