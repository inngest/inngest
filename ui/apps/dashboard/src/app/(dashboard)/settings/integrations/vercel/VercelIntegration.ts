type VercelProject = {
  id: string;
  name: string;
  servePath?: string;
  isEnabled: boolean;
};

type VercelIntegration = {
  id: string;
  name: string;
  slug: string;
  projects: VercelProject[];
  enabled: boolean;
};

export type { VercelIntegration as default };
