import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { NumberCell, TextCell } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { type Session } from '@inngest/components/types/session';
import { type LinkComponentProps } from '@tanstack/react-router';
import { createColumnHelper } from '@tanstack/react-table';

const columnHelper = createColumnHelper<Session>();

type RoutePath = LinkComponentProps['to'] | string;

type PathCreator = {
  function: (params: { functionSlug: string }) => RoutePath;
};

export function useColumns({ pathCreator }: { pathCreator: PathCreator }) {
  return [
    columnHelper.accessor('sessionId', {
      header: 'Session ID',
      enableSorting: false,
      cell: ({ row }) => (
        <TextCell>
          <span className="font-mono">{row.original.sessionId}</span>
        </TextCell>
      ),
    }),
    columnHelper.accessor('runCount', {
      header: 'Number of runs',
      enableSorting: false,
      cell: (info) => <NumberCell value={info.getValue()} term="runs" />,
    }),
    columnHelper.accessor('failedRunCount', {
      header: 'Failed runs',
      enableSorting: false,
      cell: ({ row }) => {
        const { failedRunCount, failureRate } = row.original;
        if (!failedRunCount) {
          return (
            <TextCell>
              <span className="text-light">-</span>
            </TextCell>
          );
        }
        return (
          <TextCell>
            <span>{failedRunCount}</span>{' '}
            <span className="text-tertiary-intense">{failureRate}%</span>
          </TextCell>
        );
      },
    }),
    columnHelper.accessor('lastActiveAt', {
      header: 'Last active',
      enableSorting: false,
      cell: (info) => <Time value={info.getValue()} />,
    }),
    columnHelper.accessor('functions', {
      header: 'Functions',
      enableSorting: false,
      cell: ({ row }) => (
        <HorizontalPillList
          alwaysVisibleCount={2}
          pills={row.original.functions.map((fn) => (
            <Pill
              key={fn.slug}
              appearance="outlined"
              href={pathCreator.function({ functionSlug: fn.slug })}
            >
              <PillContent type="FUNCTION">{fn.name}</PillContent>
            </Pill>
          ))}
        />
      ),
    }),
  ];
}
