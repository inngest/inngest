import { useMemo } from 'react';
import { Pill } from '@inngest/components/Pill';
import { IDCell, PillCell, TextCell, TimeCell } from '@inngest/components/Table';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { RiArrowRightSLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { AICell, EndedAtCell, RunStatusCell } from '../Table/Cell';
import type { Run, ViewScope } from './types';

const columnHelper = createColumnHelper<Run>();

const columnsIDs = [
  'app',
  'durationMS',
  'endedAt',
  'function',
  'id',
  'queuedAt',
  'startedAt',
  'status',
  'trigger',
] as const;
export type ColumnID = (typeof columnsIDs)[number];
export function isColumnID(value: unknown): value is ColumnID {
  return columnsIDs.includes(value as ColumnID);
}

// Ensure that the column ID is valid at compile time
function ensureColumnID(id: ColumnID): ColumnID {
  return id;
}

const columns = [
  columnHelper.display({
    cell: (props) => {
      const { id, status } = props.row.original;

      return (
        <div className="flex items-center">
          <RunStatusCell status={status} runID={id} />
        </div>
      );
    },
    header: 'Status',
    enableSorting: false,
    id: ensureColumnID('status'),
  }),
  columnHelper.accessor('id', {
    cell: (info) => {
      const id = info.getValue();
      const isDeferred = info.row.original.runType === 'DEFER';

      return (
        <div className="flex items-center gap-2">
          <IDCell>{id}</IDCell>
          {isDeferred && (
            <Pill appearance="outlined" kind="default">
              Defer
            </Pill>
          )}
        </div>
      );
    },
    header: 'Run ID',
    enableSorting: false,
    id: ensureColumnID('id'),
  }),
  columnHelper.display({
    cell: (props) => {
      const data = props.row.original;

      if (data.isBatch) {
        return (
          <div className="flex items-center">
            <TextCell>Batch</TextCell>
          </div>
        );
      }
      if (data.cronSchedule) {
        return (
          <div className="flex items-center">
            <PillCell type="CRON">{data.cronSchedule}</PillCell>
          </div>
        );
      }
      if (data.eventName) {
        return (
          <div className="flex items-center">
            <PillCell type="EVENT">{data.eventName}</PillCell>
          </div>
        );
      }

      return null;
    },
    header: 'Trigger',
    id: ensureColumnID('trigger'),
  }),
  columnHelper.accessor('function', {
    cell: (info) => {
      const data = info.row.original;
      const fnName = info.getValue().name;
      const isDeferred = data.runType === 'DEFER';

      const parentFunction = data.deferredFrom?.[0]?.function;
      const parentLabel = parentFunction?.name ?? parentFunction?.slug;

      const nameCell = data.hasAI ? (
        <AICell>{fnName}</AICell>
      ) : (
        <TextCell className="min-w-0">{fnName}</TextCell>
      );

      return (
        <div className="flex max-w-md items-center gap-1">
          {isDeferred && (
            <>
              <OptionalTooltip tooltip={parentLabel}>
                <span className="inline-flex">
                  <FunctionsIcon className="text-muted h-4 w-4 shrink-0" />
                </span>
              </OptionalTooltip>
              <RiArrowRightSLine className="text-muted h-4 w-4 shrink-0" />
            </>
          )}
          {nameCell}
        </div>
      );
    },
    header: 'Function',
    enableSorting: false,
    id: ensureColumnID('function'),
  }),
  columnHelper.accessor('app', {
    cell: (info) => {
      return (
        <div className="flex items-center text-nowrap">
          <TextCell>{info.getValue().externalID}</TextCell>
        </div>
      );
    },
    header: 'App',
    enableSorting: false,
    id: ensureColumnID('app'),
  }),
  columnHelper.accessor('queuedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          <TimeCell date={new Date(time)} />
        </div>
      );
    },
    header: 'Queued at',
    enableSorting: false,
    id: ensureColumnID('queuedAt'),
  }),
  columnHelper.accessor('startedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          {time ? <TimeCell date={new Date(time)} /> : <TextCell>-</TextCell>}
        </div>
      );
    },
    header: 'Started at',
    enableSorting: false,
    id: ensureColumnID('startedAt'),
  }),
  columnHelper.accessor('endedAt', {
    cell: (info) => {
      const endedAt = info.getValue();

      return <EndedAtCell runID={info.row.original.id} endedAt={endedAt} />;
    },
    header: 'Ended at',
    enableSorting: false,
    id: ensureColumnID('endedAt'),
  }),
  columnHelper.accessor('durationMS', {
    cell: (info) => {
      const duration = info.getValue();

      return (
        <div className="flex items-center">
          <TextCell>{duration ? formatMilliseconds(duration) : '-'}</TextCell>
        </div>
      );
    },
    header: 'Duration',
    enableSorting: false,
    id: ensureColumnID('durationMS'),
  }),
] as const;

/**
 * Return the correct columns for the given view scope. This is necessary to
 * avoid columns with redundant data. For example, if a user is looking at a
 * single function's runs then we shouldn't show the app or function columns
 * since every row will have the same values
 */
export function useScopedColumns(scope: ViewScope) {
  return useMemo(() => {
    return columns.filter((column) => {
      if ('accessorKey' in column) {
        if (scope === 'fn') {
          return column.accessorKey !== 'app' && column.accessorKey !== 'function';
        }
      }
      return true;
    });
  }, [scope]);
}
