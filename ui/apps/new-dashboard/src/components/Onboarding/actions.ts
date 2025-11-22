import { type InvokeFunctionMutationVariables } from "@/gql/graphql";
import { getProductionEnvironment } from "@/queries/server-only/getEnvironment";
import { createServerFn } from "@tanstack/react-start";
import {
  getInvokeFunctionLookups,
  getVercelApps,
  invokeFn,
  preloadInvokeFunctionLookups,
  syncNewApp,
  type UnattachedSync,
  type VercelApp,
} from "./data";

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
      appName: response.syncNewApp.app?.externalID || "Unknown App",
    };
  } catch (error) {
    console.error("Error syncing app:", error);
    return { success: false, error: null, appName: null };
  }
}

export async function invokeFunction({
  functionSlug,
  user,
  data,
}: Pick<InvokeFunctionMutationVariables, "data" | "functionSlug" | "user">) {
  try {
    await invokeFn({ functionSlug, user, data });

    return {
      success: true,
    };
  } catch (error) {
    console.error("Error invoking function:", error);

    if (error instanceof Error) {
      return {
        success: false,
        error: error.message,
      };
    }

    return {
      success: false,
      error: "Unknown error occurred while invoking function",
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

export type VercelSyncsResponse = {
  apps: VercelApp[];
  unattachedSyncs: UnattachedSync[];
};

export const getVercelSyncs = createServerFn({ method: "GET" }).handler(
  async () => {
    try {
      const response = await getVercelApps();
      const syncs = response.environment;
      const vercelApps = syncs.apps.filter(
        (app) => app.latestSync?.platform === "vercel" && !app.isArchived,
      );
      const unattachedSyncs = syncs.unattachedSyncs.filter(
        (sync) => sync.vercelDeploymentURL,
      );
      return { apps: vercelApps, unattachedSyncs: unattachedSyncs };
    } catch (error) {
      console.error("Error fetching vercel apps:", error);
      return { apps: [], unattachedSyncs: [] };
    }
  },
);
