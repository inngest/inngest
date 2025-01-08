'use client';

import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { cn } from '@inngest/components/utils/classNames';

import { SyncStatusPill } from '@/components/SyncStatusPill';

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
      className={cn(
        'border-muted bg-canvasBase w-2/5 max-w-2xl flex-shrink-0 overflow-y-auto border-r sm:w-1/3',
        className
      )}
    >
      {loading && (
        <div className="border-muted border-b px-4 py-3">
          <Skeleton className="mb-1 block h-11 w-full" />
        </div>
      )}
      {!loading && (
        <ul className="w-full">
          {syncs.map((sync) => {
            let bgColor = 'bg-canvasBase';
            if (sync.id === selectedSyncID) {
              bgColor = 'bg-canvasSubtle';
            }

            return (
              <li
                className={cn(
                  'border-muted text-basis hover:bg-canvasMuted flex cursor-pointer items-center justify-between border-b',
                  bgColor
                )}
                key={sync.id}
                onClick={() => onClick(sync.id)}
              >
                <div className="flex items-center gap-1">
                  <div className="ml-1 hidden w-20 items-center sm:flex">
                    <SyncStatusPill status={sync.status} />
                  </div>
                  <div className="flex-1 py-4 pl-2 align-middle sm:pl-0">
                    <Time className="text-wrap" value={sync.lastSyncedAt} />
                  </div>
                </div>
                <div
                  className="hidden items-center gap-1 align-middle md:p-2 lg:p-4 min-[1120px]:flex"
                  title={`${sync.syncedFunctions.length} ${
                    sync.syncedFunctions.length === 1 ? 'function' : 'functions'
                  }`}
                >
                  {sync.syncedFunctions.length > 0 && (
                    <>
                      <FunctionsIcon className="text-muted h-4 w-4" />
                      {sync.syncedFunctions.length}
                    </>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
