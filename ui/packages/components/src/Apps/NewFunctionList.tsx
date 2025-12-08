import { useRef } from 'react';
import { Link } from '@inngest/components/Link/NewLink';
import { Pill, PillContent } from '@inngest/components/Pill/NewPill';
import CompactPaginatedTable from '@inngest/components/Table/CompactPaginatedTable';
import type { Function } from '@inngest/components/types/function';
import type { FileRouteTypes } from '@tanstack/react-router';
import { createColumnHelper } from '@tanstack/react-table';

import { HorizontalPillList } from '../Pill/HorizontalPillList';

type Fn = Pick<Function, 'name' | 'slug' | 'triggers'>;

type Props = {
  envSlug?: string;
  functions: Fn[];
  pathCreator?: {
    function: (params: { envSlug: string; functionSlug: string }) => FileRouteTypes['to'];
    eventType: (params: { envSlug: string; eventName: string }) => FileRouteTypes['to'];
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
    function: (params: { envSlug: string; functionSlug: string }) => FileRouteTypes['to'];
    eventType: (params: { envSlug: string; eventName: string }) => FileRouteTypes['to'];
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
                to={
                  pathCreator.function({
                    envSlug,
                    functionSlug: info.row.original.slug,
                  }) as FileRouteTypes['to']
                }
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
