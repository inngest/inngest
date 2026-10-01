import { describe, expect, it } from 'vitest';

import {
  bulkPermissionLevels,
  requestedPermissionLevels,
  selectedPermissionGrants,
} from './permissionSelection';
import type { PermissionGroup, PermissionLevel } from './PermissionPicker';

const groups: PermissionGroup[] = [
  { resource: 'apps', read: ['apps:read:*'], write: ['apps:write:*'] },
  { resource: 'runs', read: ['runs:read:*'], write: [] },
];

describe('permission selection', () => {
  it.each<{
    level: PermissionLevel;
    expected: Record<string, PermissionLevel>;
  }>([
    { level: 'none', expected: { apps: 'none', runs: 'none', events: 'none' } },
    { level: 'read', expected: { apps: 'read', runs: 'read', events: 'none' } },
    {
      level: 'write',
      expected: { apps: 'write', runs: 'read', events: 'write' },
    },
  ])(
    '$level shortcut selects only available permissions',
    ({ level, expected }) => {
      const catalog = [
        ...groups,
        { resource: 'events', read: [], write: ['events:write:*'] },
      ];
      expect(bulkPermissionLevels(catalog, level)).toEqual(expected);
    },
  );

  it.each<{ level: PermissionLevel; grants: string[] }>([
    { level: 'none', grants: [] },
    { level: 'read', grants: ['apps:read:*'] },
    { level: 'write', grants: ['apps:read:*', 'apps:write:*'] },
  ])('$level grants only the selected access', ({ level, grants }) => {
    expect(selectedPermissionGrants(groups, { apps: level })).toEqual(grants);
  });

  it('ignores stale selections for resources outside the current catalog', () => {
    expect(
      selectedPermissionGrants(groups, { api_keys: 'write', apps: 'read' }),
    ).toEqual(['apps:read:*']);
  });

  it('does not restore grants excluded from a narrowed OAuth request', () => {
    const narrowed = [{ resource: 'apps', read: ['apps:read:*'], write: [] }];
    expect(selectedPermissionGrants(narrowed, { apps: 'write' })).toEqual([
      'apps:read:*',
    ]);
    expect(
      requestedPermissionLevels({
        groups: narrowed,
        scopes: ['apps:read:*', 'apps:write:*'],
      }),
    ).toEqual({ apps: 'read' });
  });

  it('preselects only requested and grantable access', () => {
    expect(
      requestedPermissionLevels({
        groups,
        scopes: ['apps:read:*', 'runs:write:*', 'api_keys:write:*'],
      }),
    ).toEqual({ apps: 'read' });
  });
});
