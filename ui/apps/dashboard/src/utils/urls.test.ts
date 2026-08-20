import { describe, expect, it } from 'vitest';

import { pathCreator } from './urls';

describe('pathCreator.functions', () => {
  it('returns the unfiltered path when no apps are given', () => {
    expect(pathCreator.functions({ envSlug: 'production' })).toBe(
      '/env/production/functions',
    );
    expect(pathCreator.functions({ envSlug: 'production', appIDs: [] })).toBe(
      '/env/production/functions',
    );
  });

  it('encodes app IDs the way the functions table parses them', () => {
    const path = pathCreator.functions({
      envSlug: 'production',
      appIDs: ['app-1', 'app-2'],
    });

    const filterApp = new URL(path, 'https://app.inngest.com').searchParams.get(
      'filterApp',
    );
    expect(filterApp).not.toBeNull();
    expect(JSON.parse(filterApp!)).toEqual(['app-1', 'app-2']);
  });

  it('sets the archived status filter for archived apps', () => {
    const params = new URL(
      pathCreator.functions({
        envSlug: 'production',
        appIDs: ['app-1'],
        archived: true,
      }),
      'https://app.inngest.com',
    ).searchParams;

    expect(params.get('archived')).toBe('true');
    expect(JSON.parse(params.get('filterApp')!)).toEqual(['app-1']);
  });

  it('omits the archived status filter for active apps', () => {
    expect(
      pathCreator.functions({
        envSlug: 'production',
        appIDs: ['app-1'],
        archived: false,
      }),
    ).not.toContain('archived');
  });
});
