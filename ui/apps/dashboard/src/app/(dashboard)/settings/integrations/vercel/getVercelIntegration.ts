import restAPI, { HTTPError } from '@/queries/restAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import { type VercelIntegration, type VercelProjectAPIResponse } from './VercelIntegration';
import enrichVercelProjects from './enrichVercelProjects';

export default async function getVercelIntegration(): Promise<VercelIntegration> {
  const environment = await getEnvironment({
    environmentSlug: 'production',
  });

  const url = new URL('/v1/integrations/vercel/projects', process.env.NEXT_PUBLIC_API_URL);
  url.searchParams.set('workspaceID', environment.id);

  let response: VercelProjectAPIResponse;
  try {
    response = await restAPI(url).json<{
      projects: { id: string; name: string }[];
    }>();
  } catch (err) {
    if (err instanceof HTTPError) {
      if (err.response.status === 400) {
        return {
          id: 'dummy-placeholder-id',
          name: 'Vercel',
          slug: 'vercel',
          projects: [],
          enabled: false,
        };
      } else if (err.response.status === 401) {
        throw new Error('Please sign in in again to view this page');
      }
    } else {
      throw err;
    }
    return {
      id: 'dummy-placeholder-id',
      name: 'Vercel',
      slug: 'vercel',
      projects: [],
      enabled: false,
    };
  }

  const projects = await enrichVercelProjects(response.projects);

  return {
    id: 'dummy-placeholder-id',
    name: 'Vercel',
    slug: 'vercel',
    projects,
    enabled: true,
  };
}
