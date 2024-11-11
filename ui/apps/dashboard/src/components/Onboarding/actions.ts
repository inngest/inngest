'use server';

import { type InvokeFunctionMutationVariables } from '@/gql/graphql';
import { getProductionEnvironment } from '@/queries/server-only/getEnvironment';
import {
  getInvokeFunctionLookups,
  getVercelApps,
  invokeFn,
  preloadInvokeFunctionLookups,
  syncNewApp,
} from './data';

export async function syncAppManually(appURL: string) {
  try {
    const response = await syncNewApp(appURL);
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
}

export async function invokeFunction({
  functionSlug,
  user,
  data,
}: Pick<InvokeFunctionMutationVariables, 'data' | 'functionSlug' | 'user'>) {
  try {
    await invokeFn({ functionSlug, user, data });

    return {
      success: true,
    };
  } catch (error) {
    console.error('Error invoking function:', error);

    if (error instanceof Error) {
      return {
        success: false,
        error: error.message,
      };
    }

    return {
      success: false,
      error: 'Unknown error occurred while invoking function',
    };
  }
}

export async function prefetchFunctions() {
  const environment = await getProductionEnvironment();

  preloadInvokeFunctionLookups(environment.slug);
  const {
    envBySlug: {
      workflows: { data: functions },
    },
  } = await getInvokeFunctionLookups(environment.slug);

  return functions;
}

export async function getVercelSyncs() {
  try {
    const response = await getVercelApps();
    const syncs = response.environment;

    // Filter apps to only include those with latestSync.platform === "vercel"
    const vercelApps = syncs.apps.filter((app) => app.latestSync?.platform === 'vercel');

    // Filter unattachedSyncs to only include those with a vercelDeploymentURL
    const unattachedSyncs = syncs.unattachedSyncs.filter((sync) => sync.vercelDeploymentURL);

    return {
      apps: vercelApps,
      unattachedSyncs: unattachedSyncs,
    };
  } catch (error) {
    console.error('Error fetching vercel apps:', error);
    return { apps: [], unattachedSyncs: [] };
  }
}
