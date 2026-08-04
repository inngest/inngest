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
  // One row per resource, with whichever actions that resource actually
  // exposes. A resource with no write endpoint shows no write toggle.
  resources: {
    name: string;
    description: string;
    read?: string;
    write?: string;
  }[];
};

/**
 * groupGrants turns the flat catalog into the shape the picker renders: grouped
 * by category, then by resource, with read and write as separate toggles on one
 * row.
 */
export function groupGrants(grants: Grant[]): GrantGroup[] {
  const byCategory = new Map<
    string,
    Map<string, GrantGroup['resources'][number]>
  >();

  for (const g of grants) {
    if (!byCategory.has(g.category)) {
      byCategory.set(g.category, new Map());
    }
    const resources = byCategory.get(g.category)!;
    const row = resources.get(g.name) ?? {
      name: g.name,
      description: g.description,
    };
    if (g.action === 'read') row.read = g.grant;
    if (g.action === 'write') row.write = g.grant;
    resources.set(g.name, row);
  }

  const ordered: GrantGroup[] = [];
  for (const category of CATEGORY_ORDER) {
    const resources = byCategory.get(category);
    if (!resources || resources.size === 0) continue;
    ordered.push({
      category,
      resources: [...resources.values()].sort((a, b) =>
        a.name.localeCompare(b.name),
      ),
    });
    byCategory.delete(category);
  }
  // Anything in a category we don't know about still renders, after the known
  // ones — better than silently hiding a permission.
  for (const [category, resources] of byCategory) {
    ordered.push({
      category,
      resources: [...resources.values()].sort((a, b) =>
        a.name.localeCompare(b.name),
      ),
    });
  }
  return ordered;
}

/**
 * readOnlyPreset returns every read grant. Full Access and Read Only both
 * enumerate grants that exist right now rather than storing a wildcard, so a
 * key minted today does not silently widen when a permission is added later.
 */
export function readOnlyPreset(grants: Grant[]): string[] {
  return grants.filter((g) => g.action === 'read').map((g) => g.grant);
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
