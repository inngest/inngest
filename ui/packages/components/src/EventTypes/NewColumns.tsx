import MiniStackedBarChart from '@inngest/components/Chart/MiniStackedBarChart';
import { HorizontalPillList } from '@inngest/components/Pill/NewHorizontalPillList';
import { Pill, PillContent } from '@inngest/components/Pill/NewPill';
import { NumberCell, TextCell } from '@inngest/components/Table';
import { type EventType } from '@inngest/components/types/eventType';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper, type Row } from '@tanstack/react-table';

import { Skeleton } from '../Skeleton';
import type { EventTypesTable } from './NewEventTypesTable';
import { useEventTypeVolume } from './useEventTypeVolume';

const columnHelper = createColumnHelper<EventType>();

const columnsIDs = ['name', 'functions', 'volume'] as const;
export type ColumnID = (typeof columnsIDs)[number];
export function isColumnID(value: unknown): value is ColumnID {
  return columnsIDs.includes(value as ColumnID);
}

//
// Ensure that the column ID is valid at compile time
//
function ensureColumnID(id: ColumnID): ColumnID {
  return id;
}

function VolumeCell({
  eventName,
  getEventTypeVolume,
}: {
  eventName: string;
  getEventTypeVolume: React.ComponentProps<typeof EventTypesTable>['getEventTypeVolume'];
}) {
  const { data, isLoading } = useEventTypeVolume(eventName, getEventTypeVolume);

  if (isLoading) return <Skeleton className="my-2 block h-3 w-48" />;
  if (!data || !data.volume) return <TextCell>—</TextCell>;

  return (
    <div className="flex items-center">
      <div className="w-16">
        <NumberCell
          value={data.volume.totalVolume}
          term={data.volume.totalVolume === 1 ? 'event' : 'events'}
        />
      </div>
      <div className="hidden md:block [&_*]:cursor-pointer">
        <MiniStackedBarChart data={data.volume.dailyVolumeSlots} />
      </div>
    </div>
  );
}

export function useColumns({
  pathCreator,
  eventTypeActions,
  getEventTypeVolume,
}: {
  pathCreator: React.ComponentProps<typeof EventTypesTable>['pathCreator'];
  eventTypeActions: React.ComponentProps<typeof EventTypesTable>['eventTypeActions'];
  getEventTypeVolume: React.ComponentProps<typeof EventTypesTable>['getEventTypeVolume'];
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
      // TODO: Re-enable this when API supports sorting by event name
      enableSorting: false,
      id: ensureColumnID('name'),
    }),
    columnHelper.accessor('functions', {
      cell: (info) => {
        const functions = info.getValue();

        if (!functions || functions.length === 0) {
          return (
            <TextCell>
              <span className="text-light">—</span>
            </TextCell>
          );
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
        const name = info.row.original.name;
        return <VolumeCell eventName={name} getEventTypeVolume={getEventTypeVolume} />;
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
