import type { PermissionGroup, PermissionLevel } from './PermissionPicker';

export function bulkPermissionLevels(
  groups: PermissionGroup[],
  level: PermissionLevel,
): Record<string, PermissionLevel> {
  return Object.fromEntries(
    groups.map((group) => [
      group.resource,
      level === 'write' && group.write.length > 0
        ? 'write'
        : level !== 'none' && group.read.length > 0
        ? 'read'
        : 'none',
    ]),
  );
}

export function selectedPermissionGrants(
  groups: PermissionGroup[],
  levels: Record<string, PermissionLevel>,
): string[] {
  return [
    ...new Set(
      groups.flatMap((group) => {
        const level = levels[group.resource];
        if (level === 'write') return [...group.read, ...group.write];
        return level === 'read' ? group.read : [];
      }),
    ),
  ].sort();
}

export function requestedPermissionLevels({
  groups,
  scopes,
}: {
  groups: PermissionGroup[];
  scopes: string[];
}): Record<string, PermissionLevel> {
  const requested = new Set(scopes);
  return Object.fromEntries(
    groups.flatMap<[string, PermissionLevel]>((group) => {
      if (group.write.some((scope) => requested.has(scope)))
        return [[group.resource, 'write']];
      if (group.read.some((scope) => requested.has(scope)))
        return [[group.resource, 'read']];
      return [];
    }),
  );
}
