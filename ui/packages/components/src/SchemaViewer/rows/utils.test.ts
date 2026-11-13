import { describe, expect, it } from 'vitest';

import { getRelativePath } from './utils';

describe('getRelativePath', () => {
  it.each([
    ['app/user.created.name', 'name'],
    ['app/user.created.data', 'data'],
    ['app/user.created.id', 'id'],
    ['app/user.created.ts', 'ts'],
    ['app/user.created.ts_dt', 'ts_dt'],
    ['app/user.created.received_at', 'received_at'],
    ['app/user.created.received_at_dt', 'received_at_dt'],
    ['app/user.created.v', 'v'],
  ])('returns the standard field for %s', (path, expected) => {
    expect(getRelativePath(path)).toBe(expected);
  });

  it.each([
    ['app/user.created.data.email', 'data.email'],
    ['app/user.created.data.items[*].qty', 'data.items[*].qty'],
    ['evt.data.plan.tier', 'data.plan.tier'],
  ])('keeps the nested path under data for %s', (path, expected) => {
    expect(getRelativePath(path)).toBe(expected);
  });

  it.each([
    // Properties inside data that are named like a standard field
    ['app/user.created.data.name', 'data.name'],
    ['app/user.created.data.id', 'data.id'],
    ['app/user.created.data.user.name', 'data.user.name'],
    ['app/user.created.data.items[*].id', 'data.items[*].id'],
    ['app/user.created.data.v', 'data.v'],
    ['app/user.created.data.meta.ts', 'data.meta.ts'],
  ])(
    'keeps the data prefix when a nested property is named like a standard field: %s',
    (path, expected) => {
      expect(getRelativePath(path)).toBe(expected);
    }
  );

  it.each([
    // "names", "ids" and "tsunami" only start with a standard field name
    ['app/user.created.data.names', 'data.names'],
    ['app/user.created.data.ids[*]', 'data.ids[*]'],
    ['app/user.created.data.version', 'data.version'],
    ['shop.tsunami.alert.data.level', 'data.level'],
    ['app.identity.verified.data.ok', 'data.ok'],
  ])('only matches whole path segments: %s', (path, expected) => {
    expect(getRelativePath(path)).toBe(expected);
  });

  it.each([['data'], ['items[*].qty'], ['app/user.created'], ['']])(
    'returns the path unchanged when there is no standard field to anchor on: %j',
    (path) => {
      expect(getRelativePath(path)).toBe(path);
    }
  );
});
