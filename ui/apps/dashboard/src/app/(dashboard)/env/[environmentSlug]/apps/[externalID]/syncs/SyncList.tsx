'use client';

import { classNames } from '@inngest/components/utils/classNames';

import { SyncStatus } from '@/components/SyncStatus';
import { Time } from '@/components/Time';

type Props = {
  className?: string;
  onClick: (syncID: string) => void;
  selectedSyncID: string;
  syncs: Sync[];
};

type Sync = {
  createdAt: Date;
  id: string;
  status: string;
  syncedFunctions: unknown[];
};

export function SyncList({ className, onClick, selectedSyncID, syncs }: Props) {
  return (
    <div className={classNames('h-full border-r border-slate-300 bg-white', className)}>
      <div className="table border-collapse">
        {syncs.map((sync) => {
          let bgColor = 'bg-white';
          if (sync.id === selectedSyncID) {
            bgColor = 'bg-slate-100';
          }

          return (
            <div
              className={classNames(
                'table-row cursor-pointer border border-r-0 border-slate-300 text-slate-800 hover:bg-slate-100',
                bgColor
              )}
              key={sync.id}
              onClick={() => onClick(sync.id)}
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
            </div>
          );
        })}
      </div>
    </div>
  );
}
