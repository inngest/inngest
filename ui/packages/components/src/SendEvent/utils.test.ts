import { describe, expect, it } from 'vitest';

import { generateCurlCode, shellQuote } from './utils';

/**
 * Minimal POSIX shell word parser: returns the single argument that a shell
 * would produce for a word made only of single-quoted segments and
 * backslash-escaped characters, which is all `shellQuote` emits.
 *
 * It is deliberately strict. Anything else left unquoted (`$`, backticks,
 * double quotes, operators, whitespace, plain text) throws, because a real
 * shell could expand, execute or split it and this parser does not model that.
 */
const parseShellWord = (word: string): string => {
  let out = '';
  let i = 0;
  while (i < word.length) {
    const char = word[i];
    if (char === "'") {
      const end = word.indexOf("'", i + 1);
      if (end === -1) throw new Error('unterminated single quote');
      out += word.slice(i + 1, end);
      i = end + 1;
    } else if (char === '\\') {
      const escaped = word[i + 1];
      if (escaped === undefined) throw new Error('trailing backslash');
      if (escaped === '\n') throw new Error('unsupported line continuation');
      out += escaped;
      i += 2;
    } else {
      throw new Error(`unsupported unquoted character: ${JSON.stringify(char)}`);
    }
  }
  return out;
};

describe('parseShellWord (test helper)', () => {
  it.each([
    ['command substitution', '$(id)'],
    ['a variable', '$HOME'],
    ['backticks', '`id`'],
    ['a double-quoted segment', '"a"'],
    ['an operator', "'a';id"],
    ['whitespace', "'a' 'b'"],
    ['plain text', 'abc'],
    ['an unterminated quote', "'abc"],
    ['a trailing backslash', "'a'\\"],
  ])('rejects %s outside single quotes', (_description, word) => {
    expect(() => parseShellWord(word)).toThrow();
  });

  it('keeps shell syntax literal inside single quotes', () => {
    expect(parseShellWord("'$(id) `id` \"x\"'\\''y'")).toBe('$(id) `id` "x"\'y');
  });
});

const dataArgument = (curl: string): string => {
  const marker = '--data ';
  return curl.slice(curl.indexOf(marker) + marker.length);
};

describe('shellQuote', () => {
  it.each([
    ['plain text', 'hello'],
    ['an apostrophe', "it's here"],
    ['several apostrophes', "'a' 'b'"],
    ['shell metacharacters', '$HOME `id` $(id) "x" \\ ! ; & |'],
    ['a newline', 'line one\nline two'],
    ['an empty string', ''],
  ])('round-trips %s as one shell word', (_description, value) => {
    expect(parseShellWord(shellQuote(value))).toBe(value);
  });

  it('leaves strings without apostrophes byte-identical inside the quotes', () => {
    expect(shellQuote('{"name":"app/user.created"}')).toBe(`'{"name":"app/user.created"}'`);
  });
});

describe('generateCurlCode', () => {
  it('keeps the existing output for payloads without apostrophes', () => {
    expect(generateCurlCode({ name: 'app/user.created', data: { id: 1 } })).toBe(
      `curl http://localhost:8288/e/dev_key \\
  -H "Content-Type: application/json" \\
  --data '{"name":"app/user.created","data":{"id":1}}'`
    );
  });

  it.each([
    ['a single event', { name: 'app/user.created', data: { message: "it's here" } }],
    [
      'a batch of events',
      [
        { name: "o'clock", data: { quote: "'" } },
        { name: 'app/plain', data: { value: '$(id)' } },
      ],
    ],
  ])(
    'passes %s containing apostrophes to the shell as one JSON argument',
    (_description, payload) => {
      const argument = parseShellWord(dataArgument(generateCurlCode(payload)));

      expect(JSON.parse(argument)).toEqual(payload);
    }
  );
});
