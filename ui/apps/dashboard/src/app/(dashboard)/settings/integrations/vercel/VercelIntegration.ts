export enum VercelDeploymentProtection {
  Disabled = '',
  ProdDeploymentURLsAndAllPreviews = 'prod_deployment_urls_and_all_previews',
  Previews = 'preview',
}

export type VercelProject = {
  id: string;
  name: string;
  servePath?: string;
  isEnabled: boolean;
  ssoProtection?: {
    deploymentType: VercelDeploymentProtection;
  };
};

export type VercelProjectViaAPI = {
  id: string;
  name: string;
  ssoProtection?: {
    deploymentType: VercelDeploymentProtection;
  };
};

export type VercelProjectAPIResponse = {
  projects: VercelProjectViaAPI[];
};

export type VercelIntegration = {
  id: string;
  name: string;
  slug: string;
  projects: VercelProject[];
  enabled: boolean;
};

export type { VercelIntegration as default };
