import { graphql } from '@/gql';

export const GetSavedVercelProjectsDocument = graphql(`
  query GetSavedVercelProjects($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      savedVercelProjects: vercelApps {
        id
        originOverride
        projectID
        protectionBypassSecret
        path
        workspaceID
        originOverride
        protectionBypassSecret
      }
    }
  }
`);
