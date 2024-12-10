import { Fragment } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { Pill } from '@inngest/components/Pill/Pill';
import { RiArrowDownSLine } from '@remixicon/react';
import dayjs from 'dayjs';
import { useQuery } from 'urql';

import { type TimeRange } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { graphql } from '@/gql';

const currentTime = new Date();

const timeRanges: TimeRange[] = [
  {
    key: '30m',
    start: new Date(currentTime.valueOf() - 30 * 60 * 1_000), // 30 Minutes
    end: currentTime,
  },
  {
    key: '60m',
    start: new Date(currentTime.valueOf() - 60 * 60 * 1_000), // 60 Minutes
    end: currentTime,
  },
  {
    key: '6h',
    start: new Date(currentTime.valueOf() - 6 * 60 * 60 * 1_000), // 6 Hours
    end: currentTime,
  },
  {
    key: '12h',
    start: new Date(currentTime.valueOf() - 12 * 60 * 60 * 1_000), // 12 Hours
    end: currentTime,
  },
  {
    key: '24h',
    start: new Date(currentTime.valueOf() - 24 * 60 * 60 * 1_000), // 24 Hours
    end: currentTime,
  },
  {
    key: '3d',
    start: new Date(currentTime.valueOf() - 3 * 24 * 60 * 60 * 1_000), // 3 Days
    end: currentTime,
  },
  {
    key: '7d',
    start: new Date(currentTime.valueOf() - 7 * 24 * 60 * 60 * 1_000), // 7 Days
    end: currentTime,
  },
  {
    key: '14d',
    start: new Date(currentTime.valueOf() - 14 * 24 * 60 * 60 * 1_000), // 14 Days
    end: currentTime,
  },
  {
    key: '30d',
    start: new Date(currentTime.valueOf() - 30 * 24 * 60 * 60 * 1_000), // 30 Days
    end: currentTime,
  },
];

export const defaultTimeRange = timeRanges[4]!;

export function getTimeRangeByKey(key: string): TimeRange | undefined {
  return timeRanges.find((timeRange) => timeRange.key === key);
}

const GetBillingPlanDocument = graphql(`
  query GetBillingPlan {
    account {
      entitlements {
        history {
          limit
        }
      }
    }
  }
`);

type DashboardTimeRangeFilterProps = {
  selectedTimeRange: TimeRange;
  onTimeRangeChange: (timeRange: TimeRange) => void;
};

// TODO: merge this with TimeRangeFilter.tsx
export default function DashboardTimeRangeFilter({
  selectedTimeRange,
  onTimeRangeChange,
}: DashboardTimeRangeFilterProps) {
  const [{ data }] = useQuery({
    query: GetBillingPlanDocument,
  });

  const logRetention = data?.account.entitlements.history.limit || 7;

  return (
    <Listbox value={selectedTimeRange} onChange={onTimeRangeChange}>
      {({ open }) => (
        <>
          <Listbox.Label className="sr-only">Filter by time</Listbox.Label>
          <div className="relative">
            <Listbox.Button className="shadow-outline-secondary-light group inline-flex items-center gap-1 rounded-[6px] bg-slate-50 px-3 py-[5px] text-sm font-medium capitalize text-slate-800 hover:bg-slate-100 focus:outline-indigo-500">
              <p>Last {getTimeRangeLabel(selectedTimeRange)}</p>
              <RiArrowDownSLine className="h-4 w-4" aria-hidden="true" />
            </Listbox.Button>

            <Transition
              show={open}
              as={Fragment}
              leave="transition ease-in duration-100"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Listbox.Options className="shadow-floating absolute right-0 z-10 mt-[5px] w-52 origin-top-right overflow-hidden rounded-md bg-white/95 py-[9px] ring-1 ring-black/5 backdrop-blur-[3px] focus:outline-none">
                {timeRanges.map((timeRange) => {
                  const timeRangeStartInDaysAgo = dayjs(currentTime).diff(
                    dayjs(timeRange.start),
                    'days'
                  );

                  const isPlanSufficient = timeRangeStartInDaysAgo <= logRetention;

                  return (
                    <Listbox.Option
                      key={timeRange.end.valueOf() - timeRange.start.valueOf()}
                      className="ui-selected:bg-indigo-100 ui-disabled:text-slate-400 ui-selected:text-indigo-700 flex cursor-pointer select-none items-center justify-between px-3.5 py-1 text-sm font-medium capitalize text-slate-800 hover:bg-slate-100 focus:outline-none"
                      value={timeRange}
                      disabled={!isPlanSufficient}
                    >
                      {getTimeRangeLabel(timeRange)}{' '}
                      {!isPlanSufficient && (
                        <Pill kind="primary" appearance="outlined">
                          Upgrade Plan
                        </Pill>
                      )}
                    </Listbox.Option>
                  );
                })}
              </Listbox.Options>
            </Transition>
          </div>
        </>
      )}
    </Listbox>
  );
}

export function getTimeRangeLabel(timeRange: TimeRange) {
  return dayjs(timeRange.start).from(timeRange.end, true);
}
