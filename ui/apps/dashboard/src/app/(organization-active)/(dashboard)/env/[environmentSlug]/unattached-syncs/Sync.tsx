'use client';

import { RiErrorWarningLine } from '@remixicon/react';

import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { SyncErrorCard } from '@/components/SyncErrorCard';
import { useSync } from './useSync';

type Props = {
  syncID: string;
};

export function Sync({ syncID }: Props) {
  const syncRes = useSync({ syncID });
  if (syncRes.error) {
    if (syncRes.error.message.includes('no rows')) {
      <div className="h-full w-full overflow-y-auto">
        <div className="mx-auto w-full max-w-[1200px] p-4">
          <div className="flex items-center gap-2.5 rounded-lg border border-red-500 bg-red-100 px-8 py-4 text-red-500">
            <RiErrorWarningLine className="h-5 w-5" />
            Sync not found
          </div>
        </div>
      </div>;
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

  const sync = syncRes.data;

  return (
    <div className="h-full w-full overflow-y-auto">
      <div className="mx-auto w-full max-w-[1200px] p-4">
        {sync.error && <SyncErrorCard className="mb-4" error={sync.error} />}

        <AppInfoCard className="mb-4" sync={sync} />
        <AppGitCard className="mb-4" sync={sync} />
      </div>
    </div>
  );
}
