import { getVercelApps, syncNewApp } from "@/components/Onboarding/data";
import type { App, Deploy } from "@/gql/graphql";
import { createServerFn } from "@tanstack/react-start";

export type VercelSyncsResponse = {
  apps: VercelApp[];
  unattachedSyncs: UnattachedSync[];
};

export type VercelApp = Pick<App, "id" | "name" | "externalID"> & {
  latestSync: Pick<
    Deploy,
    | "id"
    | "error"
    | "platform"
    | "vercelDeploymentID"
    | "vercelProjectID"
    | "status"
  > | null;
};

export type UnattachedSync = Pick<
  Deploy,
  "lastSyncedAt" | "error" | "url" | "vercelDeploymentURL"
>;

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

export const syncAppManually = createServerFn({ method: "POST" })
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
          appName: response.syncNewApp.app?.externalID || "Unknown App",
        };
      } catch (error) {
        console.error("Error syncing app:", error);
        return { success: false, error: null, appName: null };
      }
    },
  );

type CreateVercelIntegrationParams = {
  vercelAuthorizationCode: string;
};
