import { graphql } from '@/gql';

// Shared by the API keys settings page and the device-login approval page, in
// one file so graphql-codegen sees a single operation — same reason
// allowMemberKeys.ts is structured this way.
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

// Category order is fixed rather than derived, so the picker doesn't reshuffle
// when a grant is added. A category absent from the catalog renders nothing.
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
   * switches. The action is part of the identity a key actually stores, and
   * showing it that way means the row label matches what the 403 message names.
   */
  grants: Grant[];
};

// read before write within a resource: the pair reads as a widening, and it puts
// the more dangerous switch second.
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

/** groupGrants buckets the flat catalog by category, in a fixed order. */
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
  // Anything in a category we don't know about still renders, after the known
  // ones — better than silently hiding a permission.
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
 * activePreset reports which preset chip should read as selected. `custom` is a
 * derived state, not something the user picks — it lights up whenever the
 * selection stops matching a preset, which is the only honest way to label a
 * hand-edited set.
 *
 * Compared against the presets narrowed to `permitted`, so a member who picks
 * Read Only still sees Read Only selected rather than Custom.
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
 * readOnlyPreset returns every read grant except the sensitive ones. Full Access
 * and Read Only both enumerate grants that exist right now rather than storing a
 * wildcard, so a key minted today does not silently widen when a permission is
 * added later.
 *
 * Sensitive reads are excluded because they are not read-only in effect —
 * `api:key:read` returns the decrypted signing key, which is the credential the
 * SDKs authenticate with. They stay selectable; they are just never the default.
 * The server applies the same rule, so a hand-built request gains nothing.
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
 * permittedGrants returns what this caller may actually select. Admins get the
 * whole catalog; members are narrowed to the account policy. The server
 * enforces this too — this only avoids showing toggles that would be rejected.
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

/** Read Only, narrowed to what the caller may mint — the default selection. */
export function defaultSelection(
  grants: Grant[],
  permitted: Set<string>,
): string[] {
  return readOnlyPreset(grants).filter((g) => permitted.has(g));
}
