import { HorizontalPillList, Pill } from '@inngest/components/Pill';
import { type PathCreator } from '@inngest/components/SharedContext/usePathCreator';
import { TextCell, TimeCell } from '@inngest/components/Table';
import { type Event } from '@inngest/components/types/event';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper } from '@tanstack/react-table';

import { StatusDot } from '../Status/StatusDot';
import type { EventsTable } from './EventsTable';

const columnHelper = createColumnHelper<Event>();

const columnsIDs = ['name', 'runs', 'receivedAt'] as const;
export type ColumnID = (typeof columnsIDs)[number];
export function isColumnID(value: unknown): value is ColumnID {
  return columnsIDs.includes(value as ColumnID);
}

// Ensure that the column ID is valid at compile time
function ensureColumnID(id: ColumnID): ColumnID {
  return id;
}

export function useColumns({
  pathCreator,
  singleEventTypePage,
}: {
  pathCreator: PathCreator;
  singleEventTypePage: React.ComponentProps<typeof EventsTable>['singleEventTypePage'];
}) {
  const columns = [
    columnHelper.accessor('receivedAt', {
      cell: (info) => {
        const receivedAt = info.getValue();
        return <TimeCell date={receivedAt} />;
      },
      size: 200,
      header: 'Received at',
      enableSorting: false,
      id: ensureColumnID('receivedAt'),
    }),
    ...(!singleEventTypePage
      ? [
          columnHelper.accessor('name', {
            cell: (info) => {
              const name = info.getValue();
              return <TextCell>{name}</TextCell>;
            },
            size: 300,
            header: 'Event name',
            enableSorting: false,
            id: ensureColumnID('name'),
          }),
        ]
      : []),
    columnHelper.accessor('runs', {
      cell: (info) => {
        const runs = info.getValue();
        if (!runs || runs.length === 0) {
          return (
            <TextCell>
              <span className="text-light">—</span>
            </TextCell>
          );
        }

        return (
          <HorizontalPillList
            alwaysVisibleCount={2}
            pills={runs.map((run_) => (
              <Pill
                appearance="outlined"
                href={pathCreator.runPopout({ runID: run_.id })}
                key={run_.id}
              >
                <span className="flex items-center gap-1">
                  <StatusDot status={run_.status} size="small" />
                  <p
                    className={cn(
                      'truncate',
                      run_.status === 'CANCELLED' ? 'text-subtle' : 'text-basis'
                    )}
                  >
                    {run_.fnName}
                  </p>
                </span>
              </Pill>
            ))}
          />
        );
      },
      size: 500,
      header: 'Functions triggered',
      enableSorting: false,
      id: ensureColumnID('runs'),
    }),
  ];

  return columns;
}
