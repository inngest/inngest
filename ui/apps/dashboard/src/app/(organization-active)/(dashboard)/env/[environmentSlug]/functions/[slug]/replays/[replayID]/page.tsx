'use client';

import AppDetailsCard from '@inngest/components/Apps/AppDetailsCard';
import { StatusCell } from '@inngest/components/Table/Cell';
import { Time } from '@inngest/components/Time';
import { formatMilliseconds } from '@inngest/components/utils/date';

import { useGetReplay } from '@/components/Replay/useGetReplay';

type Props = {
  params: {
    replayID: string;
  };
};

export default function Page({ params }: Props) {
  const replayID = decodeURIComponent(params.replayID);
  const { data: replay, isLoading, error } = useGetReplay(replayID);
  if (!replay) return null;

  if (error) {
    throw error;
  }

  return (
    <div className="mx-auto flex h-full w-full max-w-4xl flex-col px-6 pb-4 pt-16">
      <div className="text-muted text-xs uppercase">Replay ID: {replayID}</div>
      <div className="py-1 text-2xl">{replay.name}</div>
      <StatusCell
        size="small"
        status={replay.status}
        label={replay.status === 'ENDED' ? 'Queuing complete' : 'Queuing runs'}
      />
      <AppDetailsCard title="Replay information" className="mt-9">
        <AppDetailsCard.Item
          term="Started queuing"
          detail={<Time value={replay.createdAt} />}
          loading={isLoading}
        />
        <AppDetailsCard.Item
          term="Completed queuing"
          detail={replay.endedAt ? <Time value={replay.endedAt} /> : '-'}
          loading={isLoading}
        />
        <AppDetailsCard.Item term="Queued runs" detail={replay.runsCount} loading={isLoading} />
        <AppDetailsCard.Item
          term="Skipped runs"
          detail={replay.runsSkippedCount}
          loading={isLoading}
        />
        <AppDetailsCard.Item
          term="Duration"
          detail={replay.duration ? formatMilliseconds(replay.duration) : '-'}
          loading={isLoading}
        />
        <AppDetailsCard.Item
          term="Replay from"
          detail={replay.fromRange ? <Time value={replay.fromRange} /> : '-'}
          loading={isLoading}
        />
        <AppDetailsCard.Item
          term="Replay to"
          detail={replay.toRange ? <Time value={replay.toRange} /> : '-'}
          loading={isLoading}
        />
        <AppDetailsCard.Item
          term="Filters"
          detail={
            replay.filters?.statuses?.length ? (
              <div className="flex flex-wrap gap-2">
                {replay.filters.statuses.map((status: string) => (
                  <StatusCell key={status} status={status} label={status} size="small" />
                ))}
              </div>
            ) : (
              '-'
            )
          }
          loading={isLoading}
        />
      </AppDetailsCard>
    </div>
  );
}
