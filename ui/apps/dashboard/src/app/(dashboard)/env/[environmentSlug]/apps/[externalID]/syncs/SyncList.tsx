'use client';

import Link from 'next/link';
import { classNames } from '@inngest/components/utils/classNames';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';
import { useSyncs } from './useSyncs';

type Props = {
  className?: string;
  externalAppID: string;
  selectedSyncID: string | null;
};

export function SyncList({ className, externalAppID, selectedSyncID }: Props) {
  const env = useEnvironment();

  const appRes = useSyncs({ envID: env.id, externalAppID });
  if (appRes.error) {
    throw appRes.error;
  }
  if (appRes.isLoading) {
    return null;
  }

  return (
    <div className={classNames('h-full border-r border-slate-300 bg-white', className)}>
      <div className="table border-collapse">
        {appRes.data.syncs.map((sync) => {
          let bgColor = 'bg-white';
          if (sync.id === selectedSyncID) {
            bgColor = 'bg-slate-100';
          }

          return (
            <Link
              className={classNames(
                'table-row border border-r-0 border-slate-300 hover:bg-slate-100',
                bgColor
              )}
              href={`/env/${env.slug}/apps/${externalAppID}/syncs/${sync.id}`}
              key={sync.id}
            >
              <div className="table-cell p-4 align-middle">
                <SyncStatus status={sync.status} />
              </div>
              <div className="table-cell p-4 pl-0 pr-16 align-middle">
                <Time value={sync.createdAt} />
              </div>
              <div className="table-cell whitespace-nowrap p-4 pl-0 align-middle">
                {sync.syncedFunctions.length} functions
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
