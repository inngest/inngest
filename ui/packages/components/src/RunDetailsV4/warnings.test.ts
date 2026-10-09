import { describe, expect, it } from 'vitest';

import { isWarningMetadata, type SpanMetadata } from './types';
import { getStepWarnings, summarizeWarnings } from './warnings';

const warningsMd = (values: Record<string, unknown>): SpanMetadata =>
  ({
    scope: 'step',
    kind: 'inngest.warnings',
    updatedAt: '2024-01-01T00:00:00Z',
    values,
  } as SpanMetadata);

describe('getStepWarnings', () => {
  it('returns nothing without warnings metadata', () => {
    expect(getStepWarnings(undefined)).toEqual([]);
    expect(
      getStepWarnings([
        { scope: 'step', kind: 'userland.x', updatedAt: '', values: { a: 'b' } } as SpanMetadata,
      ])
    ).toEqual([]);
  });

  it('flattens, sorts by key, and drops empty messages', () => {
    expect(getStepWarnings([warningsMd({ b: 'second', a: 'first', c: '' })])).toEqual([
      { key: 'a', message: 'first' },
      { key: 'b', message: 'second' },
    ]);
  });

  it('merges multiple entries with later ones winning', () => {
    expect(getStepWarnings([warningsMd({ a: 'old' }), warningsMd({ a: 'new', b: 'x' })])).toEqual([
      { key: 'a', message: 'new' },
      { key: 'b', message: 'x' },
    ]);
  });
});

describe('summarizeWarnings', () => {
  it('uses the message for one warning and a count for many', () => {
    expect(summarizeWarnings([{ key: 'a', message: 'hello' }])).toBe('hello');
    expect(
      summarizeWarnings([
        { key: 'a', message: 'x' },
        { key: 'b', message: 'y' },
      ])
    ).toBe('2 warnings');
  });
});

describe('getStepWarnings ordering and malformed values', () => {
  const at = (updatedAt: string, values: Record<string, unknown>): SpanMetadata => {
    return { scope: 'step', kind: 'inngest.warnings', updatedAt, values } as SpanMetadata;
  };

  it('lets the newest updatedAt win regardless of input order', () => {
    expect(
      getStepWarnings([
        at('2024-01-02T00:00:00Z', { a: 'newer' }),
        at('2024-01-01T00:00:00Z', { a: 'older' }),
      ])
    ).toEqual([{ key: 'a', message: 'newer' }]);
  });

  it('treats an empty updatedAt as oldest', () => {
    expect(
      getStepWarnings([at('2024-01-01T00:00:00Z', { a: 'dated' }), at('', { a: 'undated' })])
    ).toEqual([{ key: 'a', message: 'dated' }]);
  });

  it('skips numbers, null, objects and whitespace-only strings, and trims messages', () => {
    expect(
      getStepWarnings([
        warningsMd({
          num: 5,
          nil: null,
          obj: { x: 1 },
          blank: '   ',
          padded: '  hello  ',
        }),
      ])
    ).toEqual([{ key: 'padded', message: 'hello' }]);
  });
});

describe('getStepWarnings with both storage forms', () => {
  const legacy = (updatedAt: string, values: Record<string, unknown>): SpanMetadata => {
    return { scope: 'step', kind: 'inngest.warnings', updatedAt, values } as SpanMetadata;
  };
  const perCode = (code: string, updatedAt: string, message: unknown): SpanMetadata => {
    return {
      scope: 'step',
      kind: `inngest.warning.${code}`,
      updatedAt,
      values: { [code]: message },
    } as unknown as SpanMetadata;
  };

  it('reads the legacy form alone', () => {
    expect(getStepWarnings([legacy('2024-01-01T00:00:00Z', { a: 'old a' })])).toEqual([
      { key: 'a', message: 'old a' },
    ]);
  });

  it('reads the per-code form alone, one kind per code', () => {
    expect(
      getStepWarnings([
        perCode('b', '2024-01-01T00:00:00Z', 'new b'),
        perCode('a', '2024-01-01T00:00:00Z', 'new a'),
      ])
    ).toEqual([
      { key: 'a', message: 'new a' },
      { key: 'b', message: 'new b' },
    ]);
  });

  it('merges both forms in one step, sorted by key', () => {
    expect(
      getStepWarnings([
        legacy('2024-01-01T00:00:00Z', { c: 'legacy c', a: 'legacy a' }),
        perCode('b', '2024-01-02T00:00:00Z', 'per-code b'),
      ])
    ).toEqual([
      { key: 'a', message: 'legacy a' },
      { key: 'b', message: 'per-code b' },
      { key: 'c', message: 'legacy c' },
    ]);
  });

  it('does not duplicate a code present in both forms; the newest span wins', () => {
    expect(
      getStepWarnings([
        perCode('a', '2024-01-01T00:00:00Z', 'per-code older'),
        legacy('2024-01-02T00:00:00Z', { a: 'legacy newer' }),
      ])
    ).toEqual([{ key: 'a', message: 'legacy newer' }]);

    expect(
      getStepWarnings([
        legacy('2024-01-01T00:00:00Z', { a: 'legacy older' }),
        perCode('a', '2024-01-02T00:00:00Z', 'per-code newer'),
      ])
    ).toEqual([{ key: 'a', message: 'per-code newer' }]);
  });

  it('prefers the per-code kind for the same code when times tie or are missing', () => {
    expect(getStepWarnings([perCode('a', '', 'per-code'), legacy('', { a: 'legacy' })])).toEqual([
      { key: 'a', message: 'per-code' },
    ]);
  });

  it('does not match unrelated kinds', () => {
    expect(
      getStepWarnings([
        { scope: 'step', kind: 'inngest.warningsfoo', updatedAt: '', values: { a: 'x' } },
        { scope: 'step', kind: 'inngest.warning', updatedAt: '', values: { a: 'x' } },
        { scope: 'step', kind: 'userland.warning.a', updatedAt: '', values: { a: 'x' } },
      ] as SpanMetadata[])
    ).toEqual([]);
  });
});

describe('isWarningMetadata', () => {
  it('matches both forms and nothing else', () => {
    expect(isWarningMetadata({ kind: 'inngest.warnings' })).toBe(true);
    expect(isWarningMetadata({ kind: 'inngest.warning.dynamic_step' })).toBe(true);
    expect(isWarningMetadata({ kind: 'inngest.warningsfoo' })).toBe(false);
    expect(isWarningMetadata({ kind: 'inngest.warning' })).toBe(false);
    expect(isWarningMetadata({ kind: 'userland.warnings' })).toBe(false);
  });
});
