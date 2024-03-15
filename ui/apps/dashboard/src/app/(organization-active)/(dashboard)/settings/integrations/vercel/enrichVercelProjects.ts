import graphqlAPI from '@/queries/graphqlAPI';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';
import { type VercelProject, type VercelProjectViaAPI } from './VercelIntegration';
import mergeVercelProjectData from './mergeVercelProjectData';
import { GetSavedVercelProjectsDocument } from './queries';

/**
 * Enriches the Vercel projects with the serve paths and whether the project is enabled.
 *
 * @param vercelProjects - The Vercel projects as returned by the Vercel API.
 * @returns The enriched Vercel projects.
 */
export default async function enrichVercelProjects(
  vercelProjects: VercelProjectViaAPI[]
): Promise<VercelProject[]> {
  const environment = await getProductionEnvironment();
  const getSavedVercelProjectsResponse = await graphqlAPI.request(GetSavedVercelProjectsDocument, {
    environmentID: environment.id,
  });
  const savedVercelProjects = getSavedVercelProjectsResponse.environment.savedVercelProjects;

  return mergeVercelProjectData({
    vercelProjects,
    savedProjects: savedVercelProjects,
  });
}
