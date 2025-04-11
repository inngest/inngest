import { HorizontalPillList, Pill } from '@inngest/components/Pill';
import { TextCell, TimeCell } from '@inngest/components/Table';
import { type Event } from '@inngest/components/types/event';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper } from '@tanstack/react-table';

import { StatusDot } from '../Status/StatusDot';
import type { EventsTable } from './EventsTable';

const columnHelper = createColumnHelper<Omit<Event, 'payload'>>();

const columnsIDs = ['name', 'functions', 'receivedAt'] as const;
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
    columnHelper.accessor('functions', {
      cell: (info) => {
        const functions = info.getValue();
        if (!functions) {
          return null;
        }

        return (
          <HorizontalPillList
            alwaysVisibleCount={2}
            pills={functions.map((function_) => (
              <Pill
                appearance="outlined"
                href={pathCreator.function({ functionSlug: function_.slug })}
                key={function_.name}
              >
                <span className="flex items-center gap-1">
                  <StatusDot status={function_.status} className="h-2 w-2 shrink-0" />
                  <p
                    className={cn(
                      'truncate',
                      function_.status === 'CANCELLED' ? 'text-subtle' : 'text-basis'
                    )}
                  >
                    {function_.name}
                  </p>
                </span>
              </Pill>
            ))}
          />
        );
      },
      header: 'Functions triggered',
      enableSorting: false,
      id: ensureColumnID('functions'),
    }),
  ];

  return columns;
}
