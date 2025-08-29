'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import {
  NumberCell,
  StatusCell,
  Table,
  TableBlankState,
  TextCell,
  TimeCell,
} from '@inngest/components/Table';
import { IconReplay } from '@inngest/components/icons/Replay';
import { ReplayStatus, type Replay } from '@inngest/components/types/replay';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { useEnvironment } from '@/components/Environments/environment-context';
import { useGetReplays } from '@/components/Replay/useGetReplay';
import { pathCreator } from '@/utils/urls';

const columnHelper = createColumnHelper<Replay>();

const columns = [
  columnHelper.accessor('status', {
    header: () => 'Status',
    cell: (props) => {
      const status = props.getValue();
      return (
        <StatusCell
          status={status}
          label={status === ReplayStatus.Ended ? 'Queuing complete' : 'Queuing runs'}
        />
      );
    },
    enableSorting: false,
  }),
  columnHelper.accessor('name', {
    header: () => 'Replay name',
    cell: (props) => {
      return <TextCell>{props.getValue()}</TextCell>;
    },
    enableSorting: false,
  }),
  columnHelper.accessor('createdAt', {
    header: () => 'Started queuing',
    cell: (props) => <TimeCell date={props.getValue()} />,
    enableSorting: false,
  }),
  columnHelper.accessor('endedAt', {
    header: () => 'Completed queuing',
    cell: (props) => {
      const replayEndedAt = props.getValue();
      if (!replayEndedAt) {
        return <TextCell>-</TextCell>;
      }
      return <TimeCell date={replayEndedAt} />;
    },
    enableSorting: false,
  }),
  columnHelper.accessor('runsCount', {
    header: () => 'Queued runs',
    cell: (props) => (
      <NumberCell term={props.getValue() === 1 ? 'run' : 'runs'} value={props.getValue()} />
    ),
    enableSorting: false,
  }),
  columnHelper.accessor('runsSkippedCount', {
    header: () => 'Skipped runs',
    cell: (props) => {
      const count = props.getValue();
      if (!count) {
        return <TextCell>-</TextCell>;
      }
      return <NumberCell term={count === 1 ? 'run' : 'runs'} value={count} />;
    },
    enableSorting: false,
  }),
  columnHelper.accessor('duration', {
    header: () => 'Duration',
    cell: (props) => {
      const replayDuration = props.getValue();
      if (!replayDuration) {
        return <TextCell>-</TextCell>;
      }
      return <TextCell>{formatMilliseconds(replayDuration)}</TextCell>;
    },
    enableSorting: false,
  }),
];

type Props = {
  functionSlug: string;
};

export function ReplayList({ functionSlug }: Props) {
  const environment = useEnvironment();
  const router = useRouter();

  const { isLoading, error, data: replays } = useGetReplays(functionSlug);

  if (error) {
    throw error;
  }

  return (
    <Table
      data={replays}
      columns={columns}
      isLoading={isLoading}
      onRowClick={(row) =>
        router.push(
          pathCreator.functionReplay({
            envSlug: environment.slug,
            functionSlug,
            replayID: row.original.id,
          })
        )
      }
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
