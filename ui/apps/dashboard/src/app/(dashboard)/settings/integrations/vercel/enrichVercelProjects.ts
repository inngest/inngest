import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import type VercelIntegration from './VercelIntegration';

const GetSavedVercelProjectsDocument = graphql(`
  query GetSavedVercelProjects($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      savedVercelProjects: vercelApps {
        projectID
        path
      }
    }
  }
`);

/**
 * Enriches the Vercel projects with the serve paths and whether the project is enabled.
 *
 * @param vercelProjects - The Vercel projects as returned by the Vercel API.
 * @returns The enriched Vercel projects.
 */
export default async function enrichVercelProjects(
  vercelProjects: { id: string; name: string }[]
): Promise<VercelIntegration['projects']> {
  const environment = await getEnvironment({
    environmentSlug: 'production',
  });
  const getSavedVercelProjectsResponse = await graphqlAPI.request(GetSavedVercelProjectsDocument, {
    environmentID: environment.id,
  });
  const savedVercelProjects = getSavedVercelProjectsResponse.environment.savedVercelProjects;

  const projects = vercelProjects.map((project) => {
    const savedProject = savedVercelProjects.find(
      (savedProject) => savedProject.projectID === project.id
    );
    const isProjectEnabled = savedProject !== undefined;
    return {
      id: project.id,
      name: project.name,
      servePath: savedProject?.path ?? undefined,
      isEnabled: isProjectEnabled,
    };
  });

  // We need to sort the projects alphabetically so that the order is consistent
  const alphabeticallySortedProjects = projects.sort((a, b) => {
    if (a.name < b.name) {
      return -1;
    }
    if (a.name > b.name) {
      return 1;
    }
    return 0;
  });

  return alphabeticallySortedProjects;
}
