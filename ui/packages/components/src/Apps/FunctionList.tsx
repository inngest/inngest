import { useRef } from 'react';
import { type Route } from 'next';
import { Link } from '@inngest/components/Link';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import CompactPaginatedTable from '@inngest/components/Table/CompactPaginatedTable';
import type { Function } from '@inngest/components/types/function';
import { createColumnHelper } from '@tanstack/react-table';

type Fn = Pick<Function, 'name' | 'slug' | 'triggers'>;

type Props = {
  envSlug?: string;
  functions: Fn[];
  pathCreator?: {
    // No need to make this env agnostic, since we only want links in Cloud
    function: (params: { envSlug: string; functionSlug: string }) => Route;
    eventType: (params: { envSlug: string; eventName: string }) => Route;
  };
};

export function FunctionList({ envSlug, functions, pathCreator }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const columns = useColumns({ envSlug, pathCreator });

  const sortedFunctions = [...functions].sort((a, b) => {
    return a.name.localeCompare(b.name);
  });

  return (
    <main ref={tableContainerRef}>
      <CompactPaginatedTable columns={columns} data={sortedFunctions} />
    </main>
  );
}

function useColumns({
  envSlug,
  pathCreator,
}: {
  envSlug?: string;
  pathCreator?: {
    function: (params: { envSlug: string; functionSlug: string }) => Route;
    eventType: (params: { envSlug: string; eventName: string }) => Route;
  };
}) {
  const columnHelper = createColumnHelper<Fn>();

  return [
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();

        if (envSlug && pathCreator) {
          return (
            <div className="flex items-center">
              <Link
                size="medium"
                className="w-full text-sm"
                href={pathCreator.function({ envSlug, functionSlug: info.row.original.slug })}
              >
                {name}
              </Link>
            </div>
          );
        }

        return (
          <div className="flex items-center">
            <span className="text-basis w-full text-sm">{name}</span>
          </div>
        );
      },
      header: 'Function',
      enableSorting: false,
    }),
    columnHelper.accessor('triggers', {
      cell: (props) => {
        const triggers = props.getValue();
        return (
          <HorizontalPillList
            alwaysVisibleCount={2}
            pills={triggers.map((trigger) => {
              return (
                <Pill
                  appearance="outlined"
                  href={
                    envSlug && pathCreator && trigger.type === 'EVENT'
                      ? pathCreator.eventType({ envSlug, eventName: trigger.value })
                      : undefined
                  }
                  key={trigger.type + trigger.value}
                >
                  <PillContent type={trigger.type}>{trigger.value}</PillContent>
                </Pill>
              );
            })}
          />
        );
      },
      header: () => <span>Triggers</span>,
      enableSorting: false,
    }),
  ];
}
