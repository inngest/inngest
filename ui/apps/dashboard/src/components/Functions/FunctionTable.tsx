'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import MiniStackedBarChart from '@inngest/components/Chart/MiniStackedBarChart';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { NumberCell, TextCell } from '@inngest/components/Table';
import NewTable from '@inngest/components/Table/NewTable';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { getHumanReadableCron } from '@inngest/components/hooks/useCron';
import { type Trigger } from '@inngest/components/types/trigger';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper } from '@tanstack/react-table';

import { useEnvironment } from '@/components/Environments/environment-context';
import { FunctionTriggerTypes } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';

export type FunctionTableRow = {
  app: { name: string };
  name: string;
  isArchived: boolean;
  isPaused: boolean;
  slug: string;
  triggers: Trigger[];
  failureRate: number | undefined;
  usage:
    | {
        total: number;
        slots: { failureCount: number; startCount: number }[];
      }
    | undefined;
};

type Props = {
  rows: FunctionTableRow[] | undefined;
  isLoading?: boolean;
};

export function FunctionTable({ rows = [], isLoading }: Props) {
  const env = useEnvironment();
  const router = useRouter();

  const columns = useMemo(() => {
    return createColumns(env.slug);
  }, [env.slug]);

  return (
    <main className="bg-canvasBase flex min-h-0 flex-col overflow-y-auto">
      <NewTable
        columns={columns}
        data={rows}
        isLoading={isLoading}
        blankState={rows.length === 0 ? 'No functions' : null}
        onRowClick={(row) =>
          router.push(pathCreator.function({ envSlug: env.slug, functionSlug: row.original.slug }))
        }
        getRowHref={(row) =>
          pathCreator.function({ envSlug: env.slug, functionSlug: row.original.slug })
        }
      />
    </main>
  );
}

const columnHelper = createColumnHelper<FunctionTableRow>();

function createColumns(environmentSlug: string) {
  const columns = [
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();
        const { isPaused, isArchived } = info.row.original;

        return (
          <div className="flex items-center gap-2">
            <div
              className={cn(
                'mx-1 h-2.5 w-2.5 shrink-0 rounded-full',
                isArchived
                  ? 'bg-surfaceMuted'
                  : isPaused
                  ? 'bg-accent-subtle'
                  : 'bg-primary-moderate'
              )}
            />
            <TextCell>{name}</TextCell>
          </div>
        );
      },
      header: 'Function name',
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
                    trigger.type === FunctionTriggerTypes.Event
                      ? pathCreator.eventType({
                          envSlug: environmentSlug,
                          eventName: trigger.value,
                        })
                      : undefined
                  }
                  key={trigger.type + trigger.value}
                >
                  {trigger.type === FunctionTriggerTypes.Cron ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span>
                          <PillContent type={trigger.type}>{trigger.value}</PillContent>
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>{getHumanReadableCron(trigger.value)}</TooltipContent>
                    </Tooltip>
                  ) : (
                    <PillContent type={trigger.type}>{trigger.value}</PillContent>
                  )}
                </Pill>
              );
            })}
          />
        );
      },
      header: 'Triggers',
    }),
    columnHelper.accessor((row) => row.app.name, {
      cell: (info) => {
        const appExternalID = info.getValue();
        if (!appExternalID) {
          return null;
        }

        return (
          <div className="flex items-center">
            <Pill
              appearance="outlined"
              href={pathCreator.app({
                envSlug: environmentSlug,
                externalAppID: appExternalID,
              })}
            >
              <PillContent type="APP">{appExternalID}</PillContent>
            </Pill>
          </div>
        );
      },
      header: 'App',
    }),
    columnHelper.accessor('failureRate', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Skeleton className="my-2 block h-3 w-32" />;
        }
        if (value === 0) {
          return (
            <TextCell>
              <span className="text-light">—</span>
            </TextCell>
          );
        }

        return (
          <TextCell>
            <span className="text-tertiary-intense">{value}%</span>
          </TextCell>
        );
      },
      header: 'Failure rate (24hr)',
    }),
    columnHelper.accessor('usage', {
      cell: (info) => {
        const value = info.getValue();

        if (value === undefined) {
          return <Skeleton className="my-2 block h-3 w-32" />;
        }

        return (
          <div className="flex items-center">
            <div className="w-16">
              <NumberCell value={value.total} term={value.total === 1 ? 'function' : 'functions'} />
            </div>

            <div className="hidden lg:block [&_*]:cursor-pointer">
              <MiniStackedBarChart key="volume-chart" className="shrink-0" data={value.slots} />
            </div>
          </div>
        );
      },
      header: 'Volume (24hr)',
    }),
  ];

  return columns;
}
