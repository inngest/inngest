import { Alert } from "@inngest/components/Alert/NewAlert";
import { RiErrorWarningLine } from "@remixicon/react";

import { AppGitCard } from "@/components/Apps/AppGitCard/AppGitCard";
import { AppInfoCard } from "@/components/Apps/AppInfoCard/AppInfoCard";
import { useEnvironment } from "@/components/Environments/environment-context";
import { useSync } from "./useSync";

type Props = {
  externalAppID: string;
  syncID: string;
};

export const Sync = ({ externalAppID, syncID }: Props) => {
  const env = useEnvironment();

  const syncRes = useSync({ envID: env.id, externalAppID, syncID });
  if (syncRes.error) {
    if (syncRes.error.message.includes("no rows")) {
      return (
        <div className="h-full w-full overflow-y-auto">
          <div className="mx-auto w-full max-w-[1200px] p-4">
            <div className="border-error bg-error text-error flex items-center gap-2.5 rounded-md border px-8 py-4">
              <RiErrorWarningLine className="h-5 w-5" />
              Sync not found
            </div>
          </div>
        </div>
      );
    }
    throw syncRes.error;
  }
  if (syncRes.isLoading) {
    return (
      <div className="h-full w-full overflow-y-auto">
        <div className="mx-auto w-full max-w-[1200px] p-4">
          <AppInfoCard className="mb-4" loading />
        </div>
      </div>
    );
  }

  const { app } = syncRes.data.environment;
  const { sync } = syncRes.data;

  return (
    <div className="h-full w-full overflow-y-auto">
      <div className="mx-auto w-full max-w-[1200px] p-4">
        {sync.error && (
          <Alert className="mb-4" severity="error">
            {sync.error}
          </Alert>
        )}

        {sync.status === "duplicate" && (
          <Alert className="mb-4" severity="info">
            Function configurations have not changed since the last successful
            sync. Logic in function handlers may have changed, but they are not
            inspected when syncing.
          </Alert>
        )}

        <AppInfoCard app={app} className="mb-4" sync={sync} linkToSyncs />
        <AppGitCard className="mb-4" sync={sync} />
      </div>
    </div>
  );
};
