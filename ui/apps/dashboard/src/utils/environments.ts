import slugify from '@sindresorhus/slugify';

import { EnvironmentType, type Workspace } from '@/gql/graphql';
import { type NonEmptyArray } from '@/utils/isNonEmptyArray';

export { EnvironmentType };

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

export function getDefaultEnvironment(
  environments: NonEmptyArray<Environment>
): Environment | null {
  return environments.find((e) => e.type === EnvironmentType.Production) || null;
}

function getRecentCutOffDate(): Date {
  return new Date(new Date().valueOf() - 7 * 24 * 60 * 60 * 1000);
}

export function getSortedBranchEnvironments(
  environments: NonEmptyArray<Environment>,
  includeArchived = true
): Environment[] {
  return environments
    .filter((env) => {
      if (!includeArchived && env.isArchived) {
        return false;
      }
      return env.type === EnvironmentType.BranchChild;
    })
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
  return getSortedBranchEnvironments(environments).filter(
    (env) => new Date(env.createdAt) > cutOffDate
  );
}
export function getNonRecentBranchEnvironments(
  environments: NonEmptyArray<Environment>
): Environment[] {
  const cutOffDate = getRecentCutOffDate();
  return getSortedBranchEnvironments(environments).filter(
    (env) => new Date(env.createdAt) < cutOffDate
  );
}

// Get parent test environments created by the user, not branch envs or legacy test mode
export function getTestEnvironments(
  environments: NonEmptyArray<Environment>,
  includeArchived = true
): Environment[] {
  return environments.filter((env) => {
    if (!includeArchived && env.isArchived) {
      return false;
    }
    return env.type === EnvironmentType.Test;
  });
}

export function workspaceToEnvironment(
  workspace: Pick<
    Workspace,
    | 'id'
    | 'name'
    | 'slug'
    | 'parentID'
    | 'test'
    | 'type'
    | 'webhookSigningKey'
    | 'createdAt'
    | 'isArchived'
    | 'isAutoArchiveEnabled'
    | 'lastDeployedAt'
  >
): Environment {
  const isProduction = workspace.type === EnvironmentType.Production;

  let environmentName = workspace.name;
  if (isProduction) {
    environmentName = 'Production';
  }

  const slug = getEnvironmentSlug({
    environmentID: workspace.id,
    environmentName,
    environmentSlug: workspace.slug,
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
  environmentSlug: string | null;
  environmentType: string;
};

export function getEnvironmentSlug({
  environmentID,
  environmentName,
  environmentSlug,
  environmentType,
}: getEnvironmentSlugProps): string {
  const isProduction = environmentType === EnvironmentType.Production;

  let slug = environmentSlug || '';
  if (isProduction) {
    slug = staticSlugs.production;
  } else if (environmentType === EnvironmentType.BranchParent) {
    slug = staticSlugs.branch;
  } else if (!slug) {
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
    | 'slug'
    | 'parentID'
    | 'test'
    | 'type'
    | 'webhookSigningKey'
    | 'createdAt'
    | 'isArchived'
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
