import { defaultParseSearch } from '@tanstack/react-router';
import { describe, expect, it } from 'vitest';

import { pathCreator } from './urls';

//
// The functions table reads its filters with `useStringArraySearchParam` and
// `useSearchParam`, which both require the router to hand them a *string*.
// These tests round-trip through the router's own parser, because that parsing
// step is what a hand-rolled URLSearchParams query gets wrong: it produces
// `filterApp=["id"]`, which parses into an array and gets discarded.
//
const parse = (path: string) => {
  const [, search = ''] = path.split('?');
  return defaultParseSearch(search) as Record<string, unknown>;
};

describe('pathCreator.functions', () => {
  it('returns the unfiltered path when no filters are given', () => {
    expect(pathCreator.functions({ envSlug: 'production' })).toBe(
      '/env/production/functions',
    );
    expect(pathCreator.functions({ envSlug: 'production', appIDs: [] })).toBe(
      '/env/production/functions',
    );
  });

  it('encodes app IDs so the router yields the string the filter expects', () => {
    const search = parse(
      pathCreator.functions({
        envSlug: 'production',
        appIDs: ['app-1', 'app-2'],
      }),
    );

    // `useStringArraySearchParam` rejects anything but a string, then parses it.
    expect(typeof search.filterApp).toBe('string');
    expect(JSON.parse(search.filterApp as string)).toEqual(['app-1', 'app-2']);
  });

  it('encodes the archived filter as the string the table compares against', () => {
    const search = parse(
      pathCreator.functions({
        envSlug: 'production',
        appIDs: ['app-1'],
        archived: true,
      }),
    );

    // The table does `filteredStatus === 'true'`, so a boolean would read false.
    expect(search.archived).toBe('true');
    expect(typeof search.filterApp).toBe('string');
  });

  it('omits the archived filter for active apps', () => {
    expect(
      pathCreator.functions({
        envSlug: 'production',
        appIDs: ['app-1'],
        archived: false,
      }),
    ).not.toContain('archived');
  });
});
