import { redirect } from 'next/navigation';
import { auth } from '@clerk/nextjs';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';
import IntegrationsList from './integrations';
import type { VercelIntegration } from './vercel/VercelIntegration';
import mergeVercelProjectData from './vercel/mergeVercelProjectData';
import { GetSavedVercelProjectsDocument } from './vercel/queries';

export const notEnabledVercelIntegration: VercelIntegration = {
  id: 'not-enabled',
  name: 'Vercel',
  slug: 'vercel',
  projects: [],
  enabled: false,
};

export const vercelIntegration = async () => {
  const { getToken } = auth();
  const sessionToken = await getToken();
  const { id: environmentID } = await getProductionEnvironment();
  const {
    environment: { savedVercelProjects: savedProjects = [] },
  } = await graphqlAPI.request(GetSavedVercelProjectsDocument, {
    environmentID,
  });
  const url = new URL('/v1/integrations/vercel/projects', process.env.NEXT_PUBLIC_API_URL);
  url.searchParams.set('workspaceID', environmentID);

  const restResponse = await fetch(url, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${sessionToken}`,
    },
  });

  const data = await restResponse.json();

  const projects = mergeVercelProjectData({
    vercelProjects: data?.projects || [],
    savedProjects,
  });

  return data
    ? {
        id: 'enabled-integration-id',
        name: 'Vercel',
        slug: 'vercel',
        projects,
        enabled: true,
      }
    : notEnabledVercelIntegration;
};

export default async function IntegrationsPage() {
  const newIntegrations = await getBooleanFlag('new-integrations');
  const integration = await vercelIntegration();

  //
  // TODO: this can go away once the "new-integrations"
  // feature is fully live
  if (!newIntegrations) {
    redirect('/settings/integrations/vercel');
  }

  return <IntegrationsList integration={integration} />;
}
