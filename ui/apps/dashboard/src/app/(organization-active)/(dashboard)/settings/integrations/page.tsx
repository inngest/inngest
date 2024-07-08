import { redirect } from 'next/navigation';
import { auth } from '@clerk/nextjs';
import { z } from 'zod';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';
import IntegrationsList from './integrations';
import { VercelDeploymentProtection } from './vercel/VercelIntegration';
import mergeVercelProjectData from './vercel/mergeVercelProjectData';
import { GetSavedVercelProjectsDocument } from './vercel/queries';

const VercelProjectSchema = z.object({
  projects: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      ssoProtection: z
        .object({ deploymentType: z.nativeEnum(VercelDeploymentProtection) })
        .optional(),
    })
  ),
});

export type VercelProjectResponse = z.infer<typeof VercelProjectSchema>;

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

  if (!restResponse.ok) {
    console.log('Error calling vercel project api', restResponse.status);
    return {
      id: 'not-enabled',
      name: 'Vercel',
      slug: 'vercel',
      projects: [],
      enabled: false,
    };
  }

  const data: VercelProjectResponse = await restResponse.json();
  const parsed = VercelProjectSchema.safeParse(data);

  if (!parsed.success) {
    const e = 'Got invalid vercel project response data from api';
    console.error(e, parsed.error);
    throw new Error(e);
  }

  const projects = mergeVercelProjectData({
    vercelProjects: parsed.data.projects,
    savedProjects,
  });

  return {
    id: 'enabled-integration-id',
    name: 'Vercel',
    slug: 'vercel',
    projects,
    enabled: true,
  };
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
