'use client';

import React, { useRef } from 'react';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { ReplayStatusIcon } from '@inngest/components/ReplayStatusIcon';
import { Table } from '@inngest/components/Table';
import type { Replay } from '@inngest/components/types/replay';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';
import dayjs from 'dayjs';

import NewReplayButton from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/NewReplayButton';
import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironment } from '@/queries';
import { duration } from '@/utils/date';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

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
          totalRunCount
        }
      }
    }
  }
`);

const columnHelper = createColumnHelper<Replay>();

const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Replay Name</span>,
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
  }),
  columnHelper.accessor('createdAt', {
    header: () => <span>Created At</span>,
    cell: (props) => <Time value={props.getValue()} />,
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('endedAt', {
    header: () => <span>Ended At</span>,
    cell: (props) => {
      const replayEndedAt = props.getValue();
      if (!replayEndedAt) {
        return <span>-</span>;
      }
      return <Time value={replayEndedAt} />;
    },
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('duration', {
    header: () => <span>Duration</span>,
    cell: (props) => {
      const replayDuration = props.getValue();
      if (!replayDuration) {
        return <span>-</span>;
      }
      return <time dateTime={replayDuration.toString()}>{duration(replayDuration)}</time>;
    },
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('runsCount', {
    header: () => <span>Total Runs</span>,
    cell: (props) => props.getValue(),
    size: 250,
    minSize: 250,
  }),
];

type FunctionReplayPageProps = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};
export default function FunctionReplayPage({ params }: FunctionReplayPageProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const { data, isLoading, error } = useGraphQLQuery({
    query: GetReplaysDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug: functionSlug,
    },
    skip: !environment?.id,
  });

  const tableContainerRef = useRef<HTMLDivElement>(null);

  if (isFetchingEnvironment || isLoading) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const environmentID = data?.environment?.id;
  const functionID = data?.environment?.function?.id;

  const replays: Replay[] =
    data?.environment?.function?.replays?.map((replay) => {
      const baseReplay = {
        ...replay,
        createdAt: new Date(replay.createdAt),
        runsCount: replay.totalRunCount ?? 0,
      };

      if (replay.endedAt) {
        return {
          ...baseReplay,
          status: 'ENDED',
          endedAt: new Date(replay.endedAt),
          duration: dayjs.duration(dayjs(replay.endedAt).diff(replay.createdAt)),
        };
      }

      return {
        ...baseReplay,
        status: 'CREATED',
        endedAt: undefined, // Convert from `null` to `undefined` to match the expected type
      };
    }) ?? [];

  if (error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-5">
        <div className="inline-flex items-center gap-2 text-red-600">
          <ExclamationCircleIcon className="h-4 w-4" />
          <h2 className="text-sm">Could not load replays</h2>
        </div>
      </div>
    );
  }

  return (
    <>
      {environmentID && functionID && (
        <div className="flex items-center justify-end border-b border-slate-300 px-5 py-2">
          <NewReplayButton environmentSlug={params.environmentSlug} functionSlug={functionSlug} />
        </div>
      )}
      <div className="overflow-y-auto">
        <Table
          tableContainerRef={tableContainerRef}
          options={{
            data: replays,
            columns,
            getCoreRowModel: getCoreRowModel(),
            enableSorting: false,
          }}
          blankState={<p>You have no replays for this function.</p>}
        />
      </div>
    </>
  );
}
