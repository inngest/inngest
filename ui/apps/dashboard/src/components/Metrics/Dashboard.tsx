'use client';

import { RangePicker } from '@inngest/components/DatePicker';
import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker.jsx';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { Pill } from '@inngest/components/Pill';
import {
  useBatchedSearchParams,
  useBooleanSearchParam,
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParam';
import {
  durationToString,
  parseDuration,
  subtractDuration,
  toDate,
  type DurationType,
} from '@inngest/components/utils/date';
import { useQuery } from 'urql';

import { GetBillingPlanDocument } from '@/gql/graphql';
import { MetricsOverview } from './Overview';
import { MetricsVolume } from './Volume';
import { convert } from './utils';

export type FunctionLookup = { [id: string]: string };

export type MetricsFilters = {
  from: Date;
  until?: Date;
  selectedApps?: string[];
  selectedFns?: string[];
  autoRefresh?: boolean;
  functions: FunctionLookup;
};

export type EntityType = {
  id: string;
  name: string;
};

export const DEFAULT_DURATION = { hours: 24 };

const getFrom = (start?: Date, duration?: DurationType | '') =>
  start || subtractDuration(new Date(), duration ? duration : DEFAULT_DURATION);

const getDefaultRange = (start?: Date, end?: Date, duration?: DurationType | '') =>
  start && end
    ? {
        type: 'absolute' as const,
        start: start,
        end: end,
      }
    : {
        type: 'relative' as const,
        duration: duration ? duration : DEFAULT_DURATION,
      };

export const Dashboard = ({
  apps = [],
  functions = [],
}: {
  apps: EntityType[];
  functions: EntityType[];
}) => {
  const [selectedApps, setApps, removeApps] = useStringArraySearchParam('apps');
  const [selectedFns, setFns, removeFns] = useStringArraySearchParam('fns');
  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const [autoRefresh] = useBooleanSearchParam('autoRefresh');
  const batchUpdate = useBatchedSearchParams();

  const parsedDuration = duration && parseDuration(duration);
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  const [{ data: planData }] = useQuery({
    query: GetBillingPlanDocument,
  });

  const logRetention = Number(planData?.account.plan?.features.log_retention);
  const upgradeCutoff = subtractDuration(new Date(), { days: logRetention || 7 });

  const mappedFunctions = convert(functions);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="bg-canvasBase flex h-16 w-full flex-row items-center justify-between px-3 py-5">
        <div className="flex flex-row items-center justify-start gap-x-2">
          <EntityFilter
            type="app"
            onFilterChange={(apps) => (apps.length ? setApps(apps) : removeApps())}
            selectedEntities={selectedApps || []}
            entities={apps}
            className="h-8"
          />
          <EntityFilter
            type="function"
            onFilterChange={(fns) => (fns.length ? setFns(fns) : removeFns())}
            selectedEntities={selectedFns || []}
            entities={functions}
            className="h-8"
          />
        </div>
        <div className="flex flex-row items-center justify-end gap-x-2">
          <Pill appearance="outlined" kind="warning">
            <div className="text-nowrap">15m delay</div>
          </Pill>
          <RangePicker
            className="w-full"
            upgradeCutoff={upgradeCutoff}
            defaultValue={getDefaultRange(parsedStart, parsedEnd, parsedDuration)}
            onChange={(range: RangeChangeProps) => {
              batchUpdate({
                duration: range.type === 'relative' ? durationToString(range.duration) : null,
                start: range.type === 'absolute' ? range.start.toISOString() : null,
                end: range.type === 'absolute' ? range.end.toISOString() : null,
              });
            }}
          />
        </div>
      </div>
      <div className="bg-canvasSubtle px-6">
        <MetricsOverview
          from={getFrom(parsedStart, parsedDuration)}
          until={parsedEnd}
          selectedApps={selectedApps}
          selectedFns={selectedFns}
          autoRefresh={autoRefresh}
          functions={mappedFunctions}
        />
      </div>
      <div className="bg-canvasSubtle px-6 pb-6">
        <MetricsVolume
          from={getFrom(parsedStart, parsedDuration)}
          until={parsedEnd}
          selectedApps={selectedApps}
          selectedFns={selectedFns}
          autoRefresh={autoRefresh}
          functions={mappedFunctions}
        />
      </div>
    </div>
  );
};
