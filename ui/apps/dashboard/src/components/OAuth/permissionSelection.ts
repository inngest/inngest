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
  const grants = new Set<string>();
  for (const group of groups) {
    const level = levels[group.resource] ?? 'none';
    if (level === 'read' || level === 'write') {
      group.read.forEach((grant) => grants.add(grant));
    }
    if (level === 'write') {
      group.write.forEach((grant) => grants.add(grant));
    }
  }
  return Array.from(grants).sort();
}

export function requestedPermissionLevels({
  groups,
  scopes,
}: {
  groups: PermissionGroup[];
  scopes: string[];
}): Record<string, PermissionLevel> {
  const requested = new Set(scopes);
  const levels: Record<string, PermissionLevel> = {};
  for (const group of groups) {
    if (group.write.some((scope) => requested.has(scope))) {
      levels[group.resource] = 'write';
    } else if (group.read.some((scope) => requested.has(scope))) {
      levels[group.resource] = 'read';
    }
  }
  return levels;
}
