'use client';

import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { IconFunction } from '@inngest/components/icons/Function';
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
                <div className="flex items-center">
                  <div className="hidden w-40 p-4 align-middle lg:block">
                    <SyncStatusPill status={sync.status} />
                  </div>
                  <div className="px-2 py-4 align-middle lg:hidden">
                    <SyncStatusPill status={sync.status} iconOnly />
                  </div>
                  <div className="py-4 align-middle">
                    <Time value={sync.lastSyncedAt} />
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
                      <IconFunction className="text-subtle" />
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
