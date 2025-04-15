import MiniStackedBarChart from '@inngest/components/Chart/MiniStackedBarChart';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { NumberCell, TextCell } from '@inngest/components/Table';
import { type EventType } from '@inngest/components/types/eventType';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper, type Row } from '@tanstack/react-table';

import type { EventTypesTable } from './EventTypesTable';

const columnHelper = createColumnHelper<EventType>();

const columnsIDs = ['name', 'functions', 'volume'] as const;
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
  eventTypeActions,
}: {
  pathCreator: React.ComponentProps<typeof EventTypesTable>['pathCreator'];
  eventTypeActions: React.ComponentProps<typeof EventTypesTable>['eventTypeActions'];
}) {
  const columns = [
    columnHelper.accessor('name', {
      cell: ({ row }: { row: Row<EventType> }) => {
        const name = row.original.name;
        const archived = row.original.archived;

        return (
          <div className="flex items-center gap-2">
            <div
              className={cn(
                archived ? 'bg-surfaceMuted' : 'bg-primary-subtle',
                'mx-1 h-2.5 w-2.5 shrink-0 rounded-full'
              )}
            />
            <TextCell>{name}</TextCell>
          </div>
        );
      },
      header: 'Event name',
      maxSize: 400,
      enableSorting: true,
      id: ensureColumnID('name'),
    }),
    columnHelper.accessor('functions', {
      cell: (info) => {
        const functions = info.getValue();

        return (
          <HorizontalPillList
            alwaysVisibleCount={2}
            pills={functions.map((function_) => (
              <Pill
                appearance="outlined"
                href={pathCreator.function({ functionSlug: function_.slug })}
                key={function_.name}
              >
                <PillContent type="FUNCTION">{function_.name}</PillContent>
              </Pill>
            ))}
          />
        );
      },
      header: 'Functions triggered',
      enableSorting: false,
      id: ensureColumnID('functions'),
    }),
    columnHelper.accessor('volume', {
      cell: (info) => {
        const volume = info.getValue();

        return (
          <div className="flex items-center">
            <div className="w-16">
              <NumberCell
                value={volume.totalVolume}
                term={volume.totalVolume === 1 ? 'event' : 'events'}
              />
            </div>
            <div className="hidden md:block">
              <MiniStackedBarChart data={volume.dailyVolumeSlots} />
            </div>
          </div>
        );
      },
      header: 'Volume (24h)',
      size: 100,
      enableSorting: false,
      id: ensureColumnID('volume'),
    }),
    columnHelper.display({
      id: 'actions',
      header: undefined, // Needed to enable the iconOnly styles in the table
      size: 20,
      cell: ({ row }: { row: Row<EventType> }) => {
        return eventTypeActions(row);
      },
    }),
  ];

  return columns;
}
