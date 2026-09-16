import { describe, expect, it } from 'vitest';
import { parseEnv } from './parseEnv';
import {
  validateSecretInputs,
  validateSecretName,
  validateSecretValue,
} from './validation';

describe('pasted environment files', () => {
  it.each([
    [
      'assignments and empty values',
      'TOKEN=abc==\nEMPTY=',
      [
        { name: 'TOKEN', value: 'abc==' },
        { name: 'EMPTY', value: '' },
      ],
    ],
    [
      'comments, export, CRLF and BOM',
      '\uFEFF# test\r\nexport TOKEN = abc # comment\r\n',
      [{ name: 'TOKEN', value: 'abc' }],
    ],
    [
      'quoted spaces and hashes',
      `TOKEN="  abc#123  " # comment`,
      [{ name: 'TOKEN', value: '  abc#123  ' }],
    ],
    [
      'multiline double quotes',
      'CERT="first\nsecond"\nOTHER=value',
      [
        { name: 'CERT', value: 'first\nsecond' },
        { name: 'OTHER', value: 'value' },
      ],
    ],
    [
      'double-quoted escapes',
      'CERT="first\\nsecond\\rthird"',
      [{ name: 'CERT', value: 'first\nsecond\rthird' }],
    ],
    [
      'single-quoted literal escapes',
      "CERT='first\\nsecond'",
      [{ name: 'CERT', value: 'first\\nsecond' }],
    ],
    [
      'literal substitutions',
      'TOKEN=${OTHER}\nCOMMAND=`$(whoami)`',
      [
        { name: 'TOKEN', value: '${OTHER}' },
        { name: 'COMMAND', value: '$(whoami)' },
      ],
    ],
    [
      'exact case and non-identifier names',
      'Token=one\nTOKEN=two\n1.WITH-DOT=three',
      [
        { name: 'Token', value: 'one' },
        { name: 'TOKEN', value: 'two' },
        { name: '1.WITH-DOT', value: 'three' },
      ],
    ],
    [
      'prototype-shaped names',
      '__proto__=one\nconstructor=two',
      [
        { name: '__proto__', value: 'one' },
        { name: 'constructor', value: 'two' },
      ],
    ],
  ])('preserves %s', (_label, source, entries) => {
    expect(parseEnv(source)).toEqual({ ok: true, entries });
  });

  it.each([
    ['', 'Paste one or more'],
    ['# comment only', 'Paste one or more'],
    [
      'TOKEN=valid\nprivate-value-without-assignment',
      'Line 2: expected NAME=value.',
    ],
    ['TOKEN="private-value', 'Line 1: close the quoted value.'],
    ['TOKEN="private-value" unexpected', 'Line 1: unexpected text'],
    ['TOKEN=private-value\0', 'Line 1: Values cannot contain NUL.'],
    ['TOKEN=' + 'x'.repeat(65537), 'Line 1: Use at most 64 KiB'],
    ['TOKEN=' + 'x'.repeat(1024 * 1024), 'Paste up to 1 MiB'],
    [
      Array.from({ length: 257 }, (_, index) => `TOKEN_${index}=value`).join(
        '\n',
      ),
      'Paste up to 256',
    ],
  ])(
    'rejects invalid input without disclosing its values',
    (source, message) => {
      const result = parseEnv(source);
      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.error).toContain(message);
        expect(result.error).not.toContain('private-value');
        expect(result).not.toHaveProperty('entries');
      }
    },
  );

  it('keeps duplicates visible for review instead of silently overwriting', () => {
    expect(parseEnv('TOKEN=first\nTOKEN=second')).toEqual({
      ok: true,
      entries: [
        { name: 'TOKEN', value: 'first' },
        { name: 'TOKEN', value: 'second' },
      ],
    });
  });
});

describe('secret form validation', () => {
  it.each([
    'TOKEN',
    'Mixed Case / token',
    '1.WITH-DOT',
    '__proto__',
    'é'.repeat(128),
  ])('accepts backend-compatible name %s', (name) => {
    expect(validateSecretName(name)).toBeUndefined();
  });
  it.each([
    '',
    ' TOKEN',
    'TOKEN\u0085',
    'BAD=KEY',
    'BAD\nKEY',
    'BAD\0KEY',
    'é'.repeat(129),
  ])('rejects invalid name %s', (name) => {
    expect(validateSecretName(name)).toBeTruthy();
  });
  it('preserves empty and multiline values and counts UTF-8 bytes', () => {
    for (const value of ['', 'first\nsecond', 'é'.repeat(32768)])
      expect(validateSecretValue(value)).toBeUndefined();
    expect(validateSecretValue('é'.repeat(32769))).toBeTruthy();
  });
  it('flags every duplicate and an existing name, with exact matching', () => {
    const rows = ['TOKEN', 'TOKEN', 'EXISTING', 'existing'].map((name) => ({
      name,
      value: '',
    }));
    const errors = validateSecretInputs(rows, ['EXISTING']);
    expect(errors.map((error) => error.name)).toEqual([
      'This name appears more than once.',
      'This name appears more than once.',
      'Already saved. Replace its value from the secret list.',
      undefined,
    ]);
  });
});
