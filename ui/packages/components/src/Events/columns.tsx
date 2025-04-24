import { HorizontalPillList, Pill } from '@inngest/components/Pill';
import { TextCell, TimeCell } from '@inngest/components/Table';
import { type Event } from '@inngest/components/types/event';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper } from '@tanstack/react-table';

import { StatusDot } from '../Status/StatusDot';
import type { EventsTable } from './EventsTable';

const columnHelper = createColumnHelper<Omit<Event, 'payload'>>();

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
}: {
  pathCreator: React.ComponentProps<typeof EventsTable>['pathCreator'];
}) {
  const columns = [
    columnHelper.accessor('receivedAt', {
      cell: (info) => {
        const receivedAt = info.getValue();
        return <TimeCell date={receivedAt} />;
      },
      header: 'Received at',
      maxSize: 400,
      enableSorting: false,
      id: ensureColumnID('receivedAt'),
    }),
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();
        return <TextCell>{name}</TextCell>;
      },
      header: 'Event name',
      maxSize: 400,
      enableSorting: false,
      id: ensureColumnID('name'),
    }),
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
                  <StatusDot status={run_.status} className="h-2 w-2 shrink-0" />
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
      header: 'Functions triggered',
      enableSorting: false,
      id: ensureColumnID('runs'),
    }),
  ];

  return columns;
}
