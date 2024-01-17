'use client';

import { Skeleton } from '@inngest/components/Skeleton';
import { classNames } from '@inngest/components/utils/classNames';

import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';

type Props = {
  className?: string;
  onClick: (syncID: string) => void;
  selectedSyncID: string;
  syncs: Sync[];
  loading?: false;
};

type LoadingProps = {
  className?: string;
  onClick: (syncID: string) => void;
  selectedSyncID?: undefined;
  syncs?: Sync[];
  loading: true;
};

type Sync = {
  id: string;
  lastSyncedAt: Date;
  status: string;
  syncedFunctions: unknown[];
};

export function SyncList({
  className,
  onClick,
  selectedSyncID,
  syncs,
  loading,
}: Props | LoadingProps) {
  return (
    <div
      className={classNames(
        'w-[460px] flex-shrink-0 overflow-y-auto border-r border-slate-300 bg-white',
        className
      )}
    >
      {loading && (
        <div className="border-b border-slate-100 px-4 py-3">
          <Skeleton className="mb-1 block h-11 w-full" />
        </div>
      )}
      {!loading && (
        <ul className="w-full">
          {syncs.map((sync) => {
            let bgColor = 'bg-white';
            if (sync.id === selectedSyncID) {
              bgColor = 'bg-slate-100';
            }

            return (
              <li
                className={classNames(
                  'flex cursor-pointer items-center justify-between border-b border-slate-300 text-slate-800 hover:bg-slate-100',
                  bgColor
                )}
                key={sync.id}
                onClick={() => onClick(sync.id)}
              >
                <div className="flex items-center">
                  <div className="w-36 p-4 align-middle">
                    <SyncStatus status={sync.status} />
                  </div>
                  <div className="p-4 pl-0 align-middle">
                    <Time value={sync.lastSyncedAt} />
                  </div>
                </div>
                <div className="whitespace-nowrap p-4 pl-0 align-middle">
                  {sync.syncedFunctions.length} functions
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
