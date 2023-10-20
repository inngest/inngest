/**
 * Since the metadata is a map returned by the GraphQL api, all properties should be optional
 * Deployment metadata will depend on the platform and integration.
 */
export type DeployMetadata = {
  // Vercel deploy metadata
  id?: string;
  clientId?: string;
  createdAt?: number;
  ownerId?: string;
  payload?: VercelDeploymentPayload;
  teamId?: string;
  type?: 'deployment-ready';
  userId?: string;
  webhookId?: string;
};

type VercelDeploymentPayload = {
  deployment: {
    id: string;
    inspectorUrl: string;
    meta: VercelDeploymentMetadata | { [key: string]: string };
    name: string;
    /** URL of the deployment */
    url: string;
  };
  deploymentId: string;
  /** Vercel dashboard URL */
  links: {
    deployment: string;
    project: string;
  };
  /** Vercel billing plan */
  plan: string;
  project: string;
  projectId: string;
  regions: string[];
  target?: ('production' | 'staging') | null;
  type: 'LAMBDAS';
  url: string;
};

// This data is only present when the user is using Vercel's Github integration
type VercelDeploymentMetadata = {
  githubCommitAuthorLogin?: string;
  githubCommitAuthorName?: string;
  githubCommitMessage?: string;
  githubCommitOrg?: string;
  githubCommitRef?: string;
  githubCommitRepo?: string;
  githubCommitRepoId?: string;
  githubCommitSha?: string;
  githubDeployment?: string;
  githubOrg?: string;
  githubRepo?: string;
  githubRepoId?: string;
  githubRepoOwnerType?: string;
  githubRepoVisibility?: string;
};

export function isIntegration(metadata: DeployMetadata): boolean {
  // There is only 1 integration right now so we only need to sniff out one payload
  return metadata?.payload ? true : false;
}

type IntegrationName = 'vercel' | null;

export function getIntegrationName(metadata: DeployMetadata): IntegrationName {
  if (metadata?.payload?.links?.deployment.match(/^https:\/\/vercel.com/)) {
    return 'vercel';
  }
  return null;
}
