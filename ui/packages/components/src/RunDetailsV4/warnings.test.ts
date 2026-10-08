import { describe, expect, it } from 'vitest';

import type { SpanMetadata } from './types';
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
