import { type VercelApp } from '@/gql/graphql';
import { type VercelProject, type VercelProjectViaAPI } from './VercelIntegration';

export default function mergeVercelProjectData({
  vercelProjects = [],
  savedProjects = [],
}: {
  vercelProjects: VercelProjectViaAPI[];
  savedProjects: VercelApp[];
}): VercelProject[] {
  const projects: VercelProject[] = vercelProjects.map((project) => {
    const savedProject = savedProjects.find(
      (savedProject) => savedProject.projectID === project.id
    );
    const isProjectEnabled = savedProject !== undefined;
    return {
      id: project.id,
      name: project.name,
      servePath: savedProject?.path ?? undefined,
      isEnabled: isProjectEnabled,
      ssoProtection: project.ssoProtection,
      originOverride: savedProject?.originOverride ?? undefined,
      protectionBypassSecret: savedProject?.protectionBypassSecret ?? undefined,
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
