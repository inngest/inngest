'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/Button';
import { ReplayStatusIcon } from '@inngest/components/ReplayStatusIcon';
import { Table, TableBlankState } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { IconReplay } from '@inngest/components/icons/Replay';
import type { Replay } from '@inngest/components/types/replay';
import { differenceInMilliseconds, formatMilliseconds } from '@inngest/components/utils/date';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const GetReplaysDocument = graphql(`
  query GetReplays($environmentID: ID!, $functionSlug: String!) {
    environment: workspace(id: $environmentID) {
      id
      function: workflowBySlug(slug: $functionSlug) {
        id
        replays {
          id
          name
          createdAt
          endedAt
          functionRunsScheduledCount
        }
      }
    }
  }
`);

const columnHelper = createColumnHelper<Replay>();

const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Replay name</span>,
    cell: (props) => {
      const name = props.row.original.name;
      const status = props.row.original.status;

      return (
        <div className="flex items-center gap-2">
          <ReplayStatusIcon status={status} className="h-5 w-5" />
          <span>{name}</span>
        </div>
      );
    },
    enableSorting: false,
  }),
  columnHelper.accessor('createdAt', {
    header: () => <span>Created at</span>,
    cell: (props) => <Time value={props.getValue()} />,
    size: 250,
    minSize: 250,
    enableSorting: false,
  }),
  columnHelper.accessor('endedAt', {
    header: () => <span>Ended at</span>,
    cell: (props) => {
      const replayEndedAt = props.getValue();
      if (!replayEndedAt) {
        return <span>-</span>;
      }
      return <Time value={replayEndedAt} />;
    },
    size: 250,
    minSize: 250,
    enableSorting: false,
  }),
  columnHelper.accessor('duration', {
    header: () => <span>Duration</span>,
    cell: (props) => {
      const replayDuration = props.getValue();
      if (!replayDuration) {
        return <span>-</span>;
      }
      return <time dateTime={replayDuration.toString()}>{formatMilliseconds(replayDuration)}</time>;
    },
    size: 250,
    minSize: 250,
    enableSorting: false,
  }),
  columnHelper.accessor('runsCount', {
    header: () => <span>Runs queued</span>,
    cell: (props) => props.getValue(),
    size: 250,
    minSize: 250,
    enableSorting: false,
  }),
];

type Props = {
  functionSlug: string;
};

export function ReplayList({ functionSlug }: Props) {
  const environment = useEnvironment();
  const router = useRouter();
  const client = useClient();

  const {
    data: replays,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['replays', environment.id],
    queryFn: async () => {
      const result = await client
        .query(GetReplaysDocument, { environmentID: environment.id, functionSlug })
        .toPromise();

      if (result.error) {
        throw result.error;
      }

      // Map and transform into Replay[]
      const replays: Replay[] =
        result.data?.environment.function?.replays.map((replay) => {
          const baseReplay = {
            ...replay,
            createdAt: new Date(replay.createdAt),
            runsCount: replay.functionRunsScheduledCount,
          };

          if (replay.endedAt) {
            return {
              ...baseReplay,
              status: 'ENDED',
              endedAt: new Date(replay.endedAt),
              duration: differenceInMilliseconds(
                new Date(replay.endedAt),
                new Date(replay.createdAt)
              ),
            };
          }

          return {
            ...baseReplay,
            status: 'CREATED',
            endedAt: undefined, // Convert from `null` to `undefined` to match the expected type
          };
        }) ?? [];

      return replays;
    },
    refetchInterval: 5000,
  });

  if (error) {
    return <Alert severity="error">Could not load replays</Alert>;
  }

  return (
    <Table
      data={replays}
      columns={columns}
      isLoading={isLoading}
      blankState={
        <TableBlankState
          title="No replays found"
          icon={<IconReplay />}
          actions={
            <>
              <Button
                appearance="outlined"
                label="Refresh"
                onClick={() => router.refresh()}
                icon={<RiRefreshLine />}
                iconSide="left"
              />
              <Button
                label="Go to docs"
                href="https://inngest.com/docs/platform/replay"
                target="_blank"
                icon={<RiExternalLinkLine />}
                iconSide="left"
              />
            </>
          }
        />
      }
    />
  );
}
