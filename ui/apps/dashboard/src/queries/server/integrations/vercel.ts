import { getVercelApps, syncNewApp } from '@/components/Onboarding/data';
import {
  CreateVercelAppDocument,
  UpdateVercelAppDocument,
  RemoveVercelAppDocument,
  type App,
  type Deploy,
  type VercelApp as GraphQLVercelApp,
  type VercelIntegration as GraphQLVercelIntegration,
} from '@/gql/graphql';
import { createServerFn } from '@tanstack/react-start';
import { getProductionEnvironment } from '@/queries/server/getEnvironment';
import restAPI, { HTTPError } from '../../restAPI';
import graphqlAPI from '../../graphqlAPI';
import { graphql } from '@/gql';

import { ClientError } from 'graphql-request';

export const GetSavedVercelProjectsDocument = graphql(`
  query GetSavedVercelProjects($environmentID: ID!) {
    account {
      marketplace
    }

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

export enum VercelDeploymentProtection {
  Disabled = '',
  ProdDeploymentURLsAndAllPreviews = 'prod_deployment_urls_and_all_previews',
  Previews = 'preview',
  All = 'all',
}

export type VercelProject = {
  id: string;
  name: string;
  servePath?: string;
  isEnabled: boolean;
  ssoProtection?: {
    deploymentType: VercelDeploymentProtection;
  };
  originOverride?: string;
  protectionBypassSecret?: string;
};

export type VercelProjectViaAPI = Pick<
  VercelProject,
  'id' | 'name' | 'ssoProtection'
>;

export type VercelIntegration = {
  id: string;
  name: string;
  slug: string;
  projects: VercelProject[];
  enabled: boolean;
};

export type VercelSyncsResponse = {
  apps: VercelApp[];
  unattachedSyncs: UnattachedSync[];
};

type CreateVercelIntegrationParams = {
  vercelAuthorizationCode: string;
};

export type VercelApp = Pick<App, 'id' | 'name' | 'externalID'> & {
  latestSync: Pick<
    Deploy,
    | 'id'
    | 'error'
    | 'platform'
    | 'vercelDeploymentID'
    | 'vercelProjectID'
    | 'status'
  > | null;
};

export type UnattachedSync = Pick<
  Deploy,
  'lastSyncedAt' | 'error' | 'url' | 'vercelDeploymentURL'
>;

export const getVercelSyncs = createServerFn({ method: 'GET' }).handler(
  async () => {
    try {
      const response = await getVercelApps();
      const syncs = response.environment;
      const vercelApps = syncs.apps.filter(
        (app) => app.latestSync?.platform === 'vercel' && !app.isArchived,
      );
      const unattachedSyncs = syncs.unattachedSyncs.filter(
        (sync) => sync.vercelDeploymentURL,
      );
      return { apps: vercelApps, unattachedSyncs: unattachedSyncs };
    } catch (error) {
      console.error('Error fetching vercel apps:', error);
      return { apps: [], unattachedSyncs: [] };
    }
  },
);

export const syncAppManually = createServerFn({ method: 'POST' })
  .inputValidator((data: { appURL: string }) => data)
  .handler(
    async ({
      data,
    }): Promise<{
      success: boolean;
      error: {} | null;
      appName: string | null;
    }> => {
      try {
        const response = await syncNewApp(data.appURL);
        const error = response.syncNewApp.error;
        if (error) {
          return { success: false, error: error, appName: null };
        }
        return {
          success: true,
          error: null,
          appName: response.syncNewApp.app?.externalID || 'Unknown App',
        };
      } catch (error) {
        console.error('Error syncing app:', error);
        return { success: false, error: null, appName: null };
      }
    },
  );

export const createVercelIntegration = createServerFn({ method: 'POST' })
  .inputValidator((data: CreateVercelIntegrationParams) => data)
  .handler(async ({ data }): Promise<VercelIntegration> => {
    const environment = await getProductionEnvironment();

    try {
      const response = await restAPI
        .get('integrations/vercel/projects', {
          searchParams: {
            workspaceID: environment.id,
            code: data.vercelAuthorizationCode,
          },
        })
        .json<{
          projects: { id: string; name: string }[];
        }>();

      const projects = await enrichVercelProjectsHelper(response.projects);

      return {
        id: 'dummy-placeholder-id',
        name: 'Vercel',
        slug: 'vercel',
        projects,
        enabled: true,
      };
    } catch (error: unknown) {
      if (error instanceof HTTPError) {
        const errorBody = await error.response
          .clone()
          .text()
          .catch(() => '');

        console.error('API Error Response:', {
          status: error.response.status,
          statusText: error.response.statusText,
          body: errorBody,
          url: error.response.url,
        });
      }
      throw error;
    }
  });

//
// Helper function to merge Vercel project data with saved projects
const mergeVercelProjectDataHelper = (
  vercelProjects: VercelProjectViaAPI[],
  savedProjects: GraphQLVercelApp[],
): VercelProject[] => {
  const projects: VercelProject[] = vercelProjects.map((project) => {
    const savedProject = savedProjects.find(
      (savedProject) => savedProject.projectID === project.id,
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

  return projects.sort((a, b) => a.name.localeCompare(b.name));
};

//
// Helper function to enrich Vercel projects with saved data
const enrichVercelProjectsHelper = async (
  vercelProjects: VercelProjectViaAPI[],
): Promise<VercelProject[]> => {
  const environment = await getProductionEnvironment();
  const getSavedVercelProjectsResponse = await graphqlAPI.request(
    GetSavedVercelProjectsDocument,
    {
      environmentID: environment.id,
    },
  );
  const savedVercelProjects =
    getSavedVercelProjectsResponse.environment.savedVercelProjects;

  return mergeVercelProjectDataHelper(vercelProjects, savedVercelProjects);
};

export const enrichVercelProjects = createServerFn({ method: 'POST' })
  .inputValidator((data: { vercelProjects: VercelProjectViaAPI[] }) => data)
  .handler(
    async ({ data }): Promise<VercelProject[]> =>
      enrichVercelProjectsHelper(data.vercelProjects),
  );

export const mergeVercelProjectData = createServerFn({ method: 'POST' })
  .inputValidator(
    (data: {
      vercelProjects: VercelProjectViaAPI[];
      savedProjects: GraphQLVercelApp[];
    }) => data,
  )
  .handler(
    async ({ data }): Promise<VercelProject[]> =>
      mergeVercelProjectDataHelper(data.vercelProjects, data.savedProjects),
  );

export const updateVercelIntegration = createServerFn({ method: 'POST' })
  .inputValidator(
    (data: {
      initialIntegration: VercelIntegration;
      updatedIntegration: VercelIntegration;
    }) => data,
  )
  .handler(async ({ data }) => {
    const environment = await getProductionEnvironment();
    const { initialIntegration, updatedIntegration } = data;

    const initialProjects = initialIntegration.projects;
    const projects = updatedIntegration.projects;

    const projectsToCreate = projects.filter((project) => {
      const initialProject = initialProjects.find(
        (initialProject) => initialProject.id === project.id,
      );
      const projectIsNew = !initialProject;
      const projectHasBeenEnabled =
        !projectIsNew && !initialProject.isEnabled && project.isEnabled;
      return projectIsNew || projectHasBeenEnabled;
    });

    const projectsToUpdate = projects.filter((project) => {
      const initialProject = initialProjects.find(
        (initialProject) => initialProject.id === project.id,
      );
      return (
        initialProject &&
        project.isEnabled &&
        project.servePath !== initialProject.servePath
      );
    });

    const projectsToRemove = projects.filter((project) => {
      const initialProject = initialProjects.find(
        (initialProject) => initialProject.id === project.id,
      );
      const projectIsNew = !initialProject;
      const projectHasBeenDisabled =
        !projectIsNew && initialProject.isEnabled && !project.isEnabled;
      return !projectIsNew && projectHasBeenDisabled;
    });

    const createVercelAppPromises = projectsToCreate.map((project) =>
      graphqlAPI.request(CreateVercelAppDocument, {
        input: {
          path: project.servePath,
          projectID: project.id,
          workspaceID: environment.id,
        },
      }),
    );

    const updateVercelAppPromises = projectsToUpdate.map((project) =>
      graphqlAPI.request(UpdateVercelAppDocument, {
        input: {
          projectID: project.id,
          path: project.servePath ?? '',
        },
      }),
    );

    const removeVercelAppPromises = projectsToRemove.map((project) =>
      graphqlAPI.request(RemoveVercelAppDocument, {
        input: {
          projectID: project.id,
          workspaceID: environment.id,
        },
      }),
    );

    return Promise.all([
      ...createVercelAppPromises,
      ...updateVercelAppPromises,
      ...removeVercelAppPromises,
    ]);
  });

const vercelIntegrationQuery = graphql(`
  query VercelIntegration {
    account {
      vercelIntegration {
        isMarketplace
        projects {
          canChangeEnabled
          deploymentProtection
          isEnabled
          name
          originOverride
          projectID
          protectionBypassSecret
          servePath
        }
      }
    }
  }
`);

export const getVercelIntegration = createServerFn({
  method: 'GET',
}).handler(async (): Promise<GraphQLVercelIntegration | null> => {
  try {
    const res = await graphqlAPI.request(vercelIntegrationQuery);
    return res.account.vercelIntegration ?? null;
  } catch (err) {
    if (err instanceof ClientError) {
      const errorMessage = err.response.errors?.[0]?.message ?? 'Unknown error';

      //
      // If the Vercel access token is forbidden/expired, treat it as if there's no integration
      // this will redirect the user to the connect page
      if (errorMessage.toLowerCase().includes('forbidden')) {
        console.warn(
          'Vercel access token is forbidden, treating as no integration',
        );
        return null;
      }

      throw new Error(errorMessage);
    }
    if (err instanceof Error) {
      throw err;
    }
    throw new Error('Unknown error');
  }
});
