import slugify from '@sindresorhus/slugify';

import { EnvironmentType, type Workspace } from '@/gql/graphql';
import { type NonEmptyArray } from '@/utils/isNonEmptyArray';

export { EnvironmentType };

export const LEGACY_TEST_MODE_NAME = 'Test';

/** Environment is a "workspace" right now */
export type Environment = {
  type: EnvironmentType;
  id: string;
  hasParent: boolean;
  name: string;
  slug: string;
  webhookSigningKey: string;
  createdAt: string;
  isArchived: boolean;
  functionCount: number;
  isAutoArchiveEnabled: boolean | null | undefined;
  lastDeployedAt: string | null | undefined;
};

export function getActiveEnvironment(
  environments: NonEmptyArray<Environment>,
  environmentSlug: string
): Environment | null {
  const activeEnvironment = environments.find((e) => e.slug === environmentSlug);
  if (activeEnvironment) {
    return activeEnvironment;
  }
  return null;
}

export function getProductionEnvironment(
  environments: NonEmptyArray<Environment>
): Environment | null {
  return environments?.find((e) => e.type === EnvironmentType.Production) || null;
}

export function getLegacyTestMode(environments: NonEmptyArray<Environment>): Environment | null {
  return (
    environments?.find(
      (env) => env.type === EnvironmentType.Test && env.name === LEGACY_TEST_MODE_NAME
    ) || null
  );
}

function getRecentCutOffDate(): Date {
  return new Date(new Date().valueOf() - 7 * 24 * 60 * 60 * 1000);
}

export function getSortedBranchEnvironments(
  environments: NonEmptyArray<Environment>
): Environment[] {
  return environments
    ?.filter((env) => env.type === EnvironmentType.BranchChild)
    .sort((a, b) => {
      // Active envs are always before archived envs.
      if (!a.isArchived && b.isArchived) {
        return -1;
      }
      if (a.isArchived && !b.isArchived) {
        return 1;
      }

      // Sort by descending "last deployed" date, considering null values.
      if (a.lastDeployedAt && !b.lastDeployedAt) {
        return -1;
      }
      if (!a.lastDeployedAt && b.lastDeployedAt) {
        return 1;
      }
      if (a.lastDeployedAt && b.lastDeployedAt) {
        return new Date(a.lastDeployedAt) > new Date(b.lastDeployedAt) ? -1 : 1;
      }

      // Should be unreachable since all branch envs should have a deploy, but
      // we still need a fallback.
      return new Date(a.createdAt) > new Date(b.createdAt) ? -1 : 1;
    });
}

export function getRecentBranchEnvironments(
  environments: NonEmptyArray<Environment>
): Environment[] {
  const cutOffDate = getRecentCutOffDate();
  return getSortedBranchEnvironments(environments)?.filter(
    (env) => new Date(env.createdAt) > cutOffDate
  );
}
export function getNonRecentBranchEnvironments(
  environments: NonEmptyArray<Environment>
): Environment[] {
  const cutOffDate = getRecentCutOffDate();
  return getSortedBranchEnvironments(environments)?.filter(
    (env) => new Date(env.createdAt) < cutOffDate
  );
}

// Get parent test environments created by the user, not branch envs or legacy test mode
export function getTestEnvironments(environments: NonEmptyArray<Environment>): Environment[] {
  return environments?.filter(
    (env) => env.type === EnvironmentType.Test && env.name !== LEGACY_TEST_MODE_NAME
  );
}

export function workspaceToEnvironment(
  workspace: Pick<
    Workspace,
    | 'id'
    | 'name'
    | 'parentID'
    | 'test'
    | 'type'
    | 'webhookSigningKey'
    | 'createdAt'
    | 'isArchived'
    | 'functionCount'
    | 'isAutoArchiveEnabled'
    | 'lastDeployedAt'
  >
): Environment {
  const isProduction = workspace.type === EnvironmentType.Production;
  const isTestWorkspace = workspace.type === EnvironmentType.Test;
  const isLegacyTestMode = isTestWorkspace && workspace.name === 'default';

  let environmentName = workspace.name;
  if (isLegacyTestMode) {
    environmentName = LEGACY_TEST_MODE_NAME;
  } else if (isProduction) {
    environmentName = 'Production';
  }

  const slug = getEnvironmentSlug({
    environmentID: workspace.id,
    environmentName,
    environmentType: workspace.type,
  });

  return {
    id: workspace.id,
    name: environmentName,
    slug,
    type: workspace.type,
    hasParent: Boolean(workspace.parentID),
    webhookSigningKey: workspace.webhookSigningKey,
    createdAt: workspace.createdAt,
    isArchived: workspace.isArchived,
    functionCount: workspace.functionCount,
    isAutoArchiveEnabled: workspace.isAutoArchiveEnabled,
    lastDeployedAt: workspace.lastDeployedAt,
  };
}

export const staticSlugs = {
  production: 'production',
  branch: 'branch',
} as const;

type getEnvironmentSlugProps = {
  environmentID: string;
  environmentName: string;
  environmentType: string;
};

export function getEnvironmentSlug({
  environmentID,
  environmentName,
  environmentType,
}: getEnvironmentSlugProps): string {
  const isProduction = environmentType === EnvironmentType.Production;
  const isTestWorkspace = environmentType === EnvironmentType.Test;
  const isLegacyTestMode = isTestWorkspace && environmentName === 'default';

  let slug: string;
  if (isLegacyTestMode) {
    environmentName = LEGACY_TEST_MODE_NAME;
    slug = slugify(environmentName);
  } else if (isProduction) {
    slug = staticSlugs.production;
  } else if (environmentType === EnvironmentType.BranchParent) {
    slug = staticSlugs.branch;
  } else {
    slug = `${slugify(environmentName)}-${environmentID.split('-')[0]}`;
  }

  return slug;
}

// TEMP
// Translate existing workspaces into any "environment" like shape during the transition
export function workspacesToEnvironments(
  workspaces: Pick<
    Workspace,
    | 'id'
    | 'name'
    | 'parentID'
    | 'test'
    | 'type'
    | 'webhookSigningKey'
    | 'createdAt'
    | 'isArchived'
    | 'functionCount'
    | 'isAutoArchiveEnabled'
    | 'lastDeployedAt'
  >[]
): Environment[] {
  return workspaces.map(workspaceToEnvironment).sort((a, b) => {
    // Sort production environments first.
    if (a.type !== b.type) {
      return a.type === EnvironmentType.Production ? -1 : 1;
    }

    // Sort child environments last.
    return a.hasParent ? 1 : -1;
  });
}
