import { describe, expect, test } from 'vitest';

import {
  activePreset,
  defaultSelection,
  fullAccessPreset,
  groupGrants,
  permittedGrants,
  readOnlyPreset,
  type Grant,
} from './grants';

function grant(g: string, category: string, sensitive = false): Grant {
  const [, name, action] = g.split(':');
  return {
    grant: g,
    name: `api:${name}`,
    action,
    description: `${action} ${name}`,
    category,
    sensitive,
  };
}

const APPS = 'Apps, Functions & Runs';
const ACCOUNTS = 'Accounts, Environments & Keys';

const CATALOG: Grant[] = [
  grant('api:run:write', APPS),
  grant('api:app:read', APPS),
  grant('api:run:read', APPS),
  grant('api:env:read', ACCOUNTS),
];

describe('groupGrants', () => {
  test('orders categories by CATEGORY_ORDER, not by input order', () => {
    expect(groupGrants(CATALOG).map((g) => g.category)).toEqual([
      ACCOUNTS,
      APPS,
    ]);
  });

  test('orders rows by resource, then read before write', () => {
    const apps = groupGrants(CATALOG).find((g) => g.category === APPS);
    expect(apps?.grants.map((g) => g.grant)).toEqual([
      'api:app:read',
      'api:run:read',
      'api:run:write',
    ]);
  });

  test('renders an unknown category rather than hiding it', () => {
    const withUnknown = [...CATALOG, grant('api:future:read', 'Warp Drive')];
    expect(groupGrants(withUnknown).map((g) => g.category)).toEqual([
      ACCOUNTS,
      APPS,
      'Warp Drive',
    ]);
  });
});

describe('activePreset', () => {
  const all = new Set(CATALOG.map((g) => g.grant));

  test('every read grant reads as Read only', () => {
    const selected = ['api:app:read', 'api:env:read', 'api:run:read'];
    expect(activePreset(selected, CATALOG, all)).toBe('readOnly');
  });

  test('the whole catalog reads as Full access', () => {
    const selected = CATALOG.map((g) => g.grant);
    expect(activePreset(selected, CATALOG, all)).toBe('fullAccess');
  });

  test('one extra grant beyond Read only reads as Custom', () => {
    const selected = ['api:app:read', 'api:env:read', 'api:run:write'];
    expect(activePreset(selected, CATALOG, all)).toBe('custom');
  });

  // A member's Read only is narrower than an admin's. Comparing against the
  // full preset would label their own preset click "Custom".
  test('a member picking Read only still reads as Read only', () => {
    const policy = {
      enabled: true,
      allowProduction: false,
      grants: ['api:run:read'],
    };
    const permitted = permittedGrants(CATALOG, policy, false);
    const selected = defaultSelection(CATALOG, permitted);
    expect(selected).toEqual(['api:run:read']);
    expect(activePreset(selected, CATALOG, permitted)).toBe('readOnly');
  });

  test('an empty selection is Custom, not a preset', () => {
    expect(activePreset([], CATALOG, all)).toBe('custom');
  });
});

// api:key:read returns the decrypted signing key, so a "read only" key holding
// it can do everything the write grants gate. It stays selectable, but is never
// handed out unasked.
describe('sensitive grants', () => {
  const SENSITIVE: Grant[] = [
    ...CATALOG,
    grant('api:key:read', ACCOUNTS, true),
  ];
  const all = new Set(SENSITIVE.map((g) => g.grant));

  test('Read only leaves them out', () => {
    expect(readOnlyPreset(SENSITIVE)).toEqual([
      'api:app:read',
      'api:run:read',
      'api:env:read',
    ]);
  });

  test('Full access still includes them — it means everything', () => {
    expect(fullAccessPreset(SENSITIVE)).toContain('api:key:read');
  });

  test('they are not preselected, even for an admin', () => {
    const permitted = permittedGrants(SENSITIVE, undefined, true);
    expect(permitted.has('api:key:read')).toBe(true);
    expect(defaultSelection(SENSITIVE, permitted)).not.toContain(
      'api:key:read',
    );
  });

  test('adding one on top of Read only reads as Custom', () => {
    const selected = [...readOnlyPreset(SENSITIVE), 'api:key:read'];
    expect(activePreset(selected, SENSITIVE, all)).toBe('custom');
  });
});

describe('permittedGrants', () => {
  test('an admin may select the whole catalog', () => {
    expect(permittedGrants(CATALOG, undefined, true).size).toBe(CATALOG.length);
  });

  test('a policy grant that is not in the catalog is ignored', () => {
    const policy = {
      enabled: true,
      allowProduction: false,
      grants: ['api:run:read', 'api:retired:read'],
    };
    expect([...permittedGrants(CATALOG, policy, false)]).toEqual([
      'api:run:read',
    ]);
  });

  test('a member with no policy may select nothing', () => {
    expect(permittedGrants(CATALOG, undefined, false).size).toBe(0);
  });
});
