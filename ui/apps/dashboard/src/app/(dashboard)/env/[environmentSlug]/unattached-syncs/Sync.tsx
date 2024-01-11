'use client';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { AppGitCard } from '@/components/AppGitCard/AppGitCard';
import { AppInfoCard } from '@/components/AppInfoCard';
import { useSync } from './useSync';

type Props = {
  syncID: string;
};

export function Sync({ syncID }: Props) {
  const syncRes = useSync({ syncID });
  if (syncRes.error) {
    if (syncRes.error.message.includes('no rows')) {
      // TODO: Make pretty
      return <div>Sync not found</div>;
    }
    throw syncRes.error;
  }
  if (syncRes.isLoading) {
    // TODO: Make pretty
    return 'Loading...';
  }

  const sync = syncRes.data;

  return (
    <div className="flex w-full justify-center p-4">
      <div className="w-full max-w-[1200px]">
        <AppInfoCard className="mb-4" sync={sync} />
        <AppGitCard className="mb-4" sync={sync} />
      </div>
    </div>
  );
}
