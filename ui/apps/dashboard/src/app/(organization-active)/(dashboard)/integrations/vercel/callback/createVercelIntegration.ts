import type VercelIntegration from '@/app/(organization-active)/(dashboard)/settings/integrations/vercel/VercelIntegration';
import enrichVercelProjects from '@/app/(organization-active)/(dashboard)/settings/integrations/vercel/enrichVercelProjects';
import restAPI from '@/queries/restAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';

type CreateVercelIntegrationParams = {
  vercelAuthorizationCode: string;
};

export default async function createVercelIntegration({
  vercelAuthorizationCode,
}: CreateVercelIntegrationParams): Promise<VercelIntegration> {
  const environment = await getProductionEnvironment();

  const url = new URL('/v1/integrations/vercel/projects', process.env.NEXT_PUBLIC_API_URL);
  url.searchParams.set('workspaceID', environment.id);
  url.searchParams.set('code', vercelAuthorizationCode);

  const response = await restAPI(url).json<{
    projects: { id: string; name: string }[];
  }>();

  const projects = await enrichVercelProjects(response.projects);

  return {
    id: 'dummy-placeholder-id',
    name: 'Vercel',
    slug: 'vercel',
    projects,
    enabled: true,
  };
}
