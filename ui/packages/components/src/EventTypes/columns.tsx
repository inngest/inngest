import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { TextCell } from '@inngest/components/Table';
import { type EventType } from '@inngest/components/types/eventType';
import { createColumnHelper, type Row } from '@tanstack/react-table';

import { ActionsMenu } from './ActionsMenu';

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

export function useColumns() {
  const columns = [
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();

        return (
          <div className="flex items-center">
            <TextCell>{name}</TextCell>
          </div>
        );
      },
      header: 'Event name',
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
                // href={pathCreator.function({
                //   envSlug: env.slug,
                //   functionSlug: function_.slug,
                // })}
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
            {volume.totalVolume}
            {volume.chart}
          </div>
        );
      },
      header: 'Volume (24h)',
      enableSorting: false,
      id: ensureColumnID('volume'),
    }),
    columnHelper.display({
      id: 'actions',
      header: () => null,
      size: 20,
      cell: ({ row }: { row: Row<EventType> }) => {
        return <ActionsMenu isArchived={row.original.archived} />;
      },
    }),
  ];

  return columns;
}
