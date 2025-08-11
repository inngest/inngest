import MiniStackedBarChart from '@inngest/components/Chart/MiniStackedBarChart';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { NumberCell, TextCell } from '@inngest/components/Table';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { getHumanReadableCron } from '@inngest/components/hooks/useCron';
import { type Function } from '@inngest/components/types/function';
import { TriggerTypes } from '@inngest/components/types/trigger';
import { cn } from '@inngest/components/utils/classNames';
import { createColumnHelper, type Row } from '@tanstack/react-table';

import { FunctionsTable } from './FunctionsTable';
import { useFunctionVolume } from './useFunctionVolume';

const columnHelper = createColumnHelper<Function>();

const columnsIDs = ['name', 'functions', 'usage'] as const;
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
  getFunctionVolume,
}: {
  pathCreator: React.ComponentProps<typeof FunctionsTable>['pathCreator'];
  getFunctionVolume: React.ComponentProps<typeof FunctionsTable>['getFunctionVolume'];
}) {
  const columns = [
    columnHelper.accessor('name', {
      cell: ({ row }: { row: Row<Function> }) => {
        const name = row.original.name;
        const { isPaused, isArchived } = row.original;

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
      enableSorting: false,
      id: ensureColumnID('name'),
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
                    trigger.type === TriggerTypes.Event
                      ? pathCreator.eventType({
                          eventName: trigger.value,
                        })
                      : undefined
                  }
                  key={trigger.type + trigger.value}
                >
                  {trigger.type === TriggerTypes.Cron ? (
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
    columnHelper.accessor('app', {
      cell: (info) => {
        const appExternalID = info.getValue()?.externalID;
        if (!appExternalID) {
          return null;
        }

        return (
          <div className="flex items-center">
            <Pill
              appearance="outlined"
              href={pathCreator.app({
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
        const functionID = info.row.original.id;
        const { data, isLoading } = useFunctionVolume(functionID, getFunctionVolume);
        if (isLoading) {
          return <Skeleton className="my-2 block h-3 w-32" />;
        }
        if (data?.failureRate === 0 || !data?.failureRate) {
          return (
            <TextCell>
              <span className="text-light">—</span>
            </TextCell>
          );
        }

        return (
          <TextCell>
            <span className="text-tertiary-intense">{data.failureRate}%</span>
          </TextCell>
        );
      },
      header: 'Failure rate (24hr)',
    }),

    columnHelper.accessor('usage', {
      cell: (info) => {
        const functionID = info.row.original.id;
        const { data, isLoading } = useFunctionVolume(functionID, getFunctionVolume);

        if (isLoading) return <Skeleton className="my-2 block h-3 w-48" />;
        if (!data || !data.usage) return <TextCell>—</TextCell>;

        return (
          <div className="flex items-center">
            <div className="w-16">
              <NumberCell
                value={data.usage.totalVolume}
                term={data.usage.totalVolume === 1 ? 'run' : 'runs'}
              />
            </div>
            <div className="hidden md:block [&_*]:cursor-pointer">
              <MiniStackedBarChart data={data.usage.dailyVolumeSlots} />
            </div>
          </div>
        );
      },
      header: 'Volume (24h)',
      enableSorting: false,
      id: ensureColumnID('usage'),
    }),
  ];

  return columns;
}
