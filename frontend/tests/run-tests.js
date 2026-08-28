import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc';
import { centsToDecimalString, formatCurrencyFromCents } from '../src/utils/currency.js';

assert.equal(centsToDecimalString(1), '0.01');
assert.equal(centsToDecimalString(10), '0.10');
assert.equal(centsToDecimalString(30), '0.30');
assert.equal(centsToDecimalString(12345), '123.45');
assert.equal(centsToDecimalString(-12345), '-123.45');
assert.equal(centsToDecimalString(Number.MAX_SAFE_INTEGER + 1), '');

assert.equal(formatCurrencyFromCents(30, 'USD'), '$0.30');
assert.equal(formatCurrencyFromCents(12345, 'USD'), '$123.45');
assert.equal(formatCurrencyFromCents(12345, 'BRL'), 'R$123.45');

const testsDirectory = path.dirname(fileURLToPath(import.meta.url));
const componentDirectory = path.resolve(testsDirectory, '../src/components');
const componentFiles = readdirSync(componentDirectory)
  .filter(fileName => fileName.endsWith('.vue'));

for (const fileName of componentFiles) {
  const filePath = path.join(componentDirectory, fileName);
  const source = readFileSync(filePath, 'utf8');
  const { descriptor, errors: parseErrors } = parse(source, { filename: filePath });

  assert.deepEqual(parseErrors, [], `${fileName} contains parse errors`);
  if (descriptor.script || descriptor.scriptSetup) {
    compileScript(descriptor, { id: filePath });
  }
  if (descriptor.template) {
    const result = compileTemplate({
      id: filePath,
      filename: filePath,
      source: descriptor.template.content,
    });
    assert.deepEqual(result.errors, [], `${fileName} contains template compilation errors`);
  }
}

console.log(`Frontend tests passed: exact cent formatting and ${componentFiles.length} Vue components verified.`);
