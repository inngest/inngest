import { graphql } from '@/gql';

// One file so graphql-codegen sees a single operation, shared by the API keys
// page and the device-login page.
export const APIKeyGrantsQuery = graphql(`
  query GetAPIKeyGrants {
    apiKeyGrants {
      grant
      name
      action
      description
      category
      sensitive
    }
    account {
      memberAPIKeyPolicy {
        enabled
        allowProduction
        grants
      }
    }
  }
`);

export const SetMemberAPIKeyPolicyMutation = graphql(`
  mutation SetMemberAPIKeyPolicy($input: MemberAPIKeyPolicyInput!) {
    setMemberAPIKeyPolicy(input: $input) {
      enabled
      allowProduction
      grants
    }
  }
`);

export type Grant = {
  grant: string;
  name: string;
  action: string;
  description: string;
  category: string;
  sensitive: boolean;
};

export type MemberPolicy = {
  enabled: boolean;
  allowProduction: boolean;
  grants: string[];
};

// Fixed rather than derived so the picker doesn't reshuffle when a grant is
// added. A category absent from the catalog renders nothing.
export const CATEGORY_ORDER = [
  'Accounts, Environments & Keys',
  'Apps, Functions & Runs',
  'Observability & AI Evals',
  'Compute',
] as const;

export type GrantGroup = {
  category: string;
  /**
   * One row per grant, so `api:env:read` and `api:env:write` are separate
   * switches and each row label matches what a 403 names.
   */
  grants: Grant[];
};

// Read before write within a resource, so the more dangerous switch is second.
const ACTION_ORDER = ['read', 'write'];

function compareGrants(a: Grant, b: Grant): number {
  if (a.name !== b.name) return a.name.localeCompare(b.name);
  const ai = ACTION_ORDER.indexOf(a.action);
  const bi = ACTION_ORDER.indexOf(b.action);
  return (
    (ai === -1 ? ACTION_ORDER.length : ai) -
    (bi === -1 ? ACTION_ORDER.length : bi)
  );
}

export function groupGrants(grants: Grant[]): GrantGroup[] {
  const byCategory = new Map<string, Grant[]>();
  for (const g of grants) {
    const list = byCategory.get(g.category);
    if (list) {
      list.push(g);
    } else {
      byCategory.set(g.category, [g]);
    }
  }

  const ordered: GrantGroup[] = [];
  const take = (category: string) => {
    const list = byCategory.get(category);
    if (!list?.length) return;
    ordered.push({ category, grants: [...list].sort(compareGrants) });
    byCategory.delete(category);
  };

  for (const category of CATEGORY_ORDER) take(category);
  // Unknown categories still render, after the known ones, so a permission is
  // never silently hidden.
  for (const category of [...byCategory.keys()]) take(category);
  return ordered;
}

export type PresetName = 'readOnly' | 'fullAccess' | 'custom';

function sameSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const set = new Set(a);
  return b.every((g) => set.has(g));
}

/**
 * `custom` is derived, not chosen: it means the selection no longer matches a
 * preset. Presets are narrowed to `permitted` first, so a member who picks Read
 * Only sees Read Only rather than Custom.
 */
export function activePreset(
  selected: string[],
  grants: Grant[],
  permitted: Set<string>,
): PresetName {
  const narrow = (preset: string[]) => preset.filter((g) => permitted.has(g));
  if (sameSet(selected, narrow(readOnlyPreset(grants)))) return 'readOnly';
  if (sameSet(selected, narrow(fullAccessPreset(grants)))) return 'fullAccess';
  return 'custom';
}

/**
 * Both presets enumerate the grants that exist now rather than storing a
 * wildcard, so a key minted today does not widen when a permission is added.
 *
 * Sensitive reads are excluded because they are not read-only in effect:
 * `api:key:read` returns the decrypted signing key. They stay selectable, just
 * never preselected. The server applies the same rule.
 */
export function readOnlyPreset(grants: Grant[]): string[] {
  return grants
    .filter((g) => g.action === 'read' && !g.sensitive)
    .map((g) => g.grant);
}

export function fullAccessPreset(grants: Grant[]): string[] {
  return grants.map((g) => g.grant);
}

/**
 * The server enforces this too. Narrowing here only avoids showing toggles that
 * would be rejected.
 */
export function permittedGrants(
  grants: Grant[],
  policy: MemberPolicy | undefined,
  isAdmin: boolean,
): Set<string> {
  if (isAdmin) return new Set(grants.map((g) => g.grant));
  const permitted = new Set(policy?.grants ?? []);
  return new Set(
    grants.filter((g) => permitted.has(g.grant)).map((g) => g.grant),
  );
}

export function defaultSelection(
  grants: Grant[],
  permitted: Set<string>,
): string[] {
  return readOnlyPreset(grants).filter((g) => permitted.has(g));
}
