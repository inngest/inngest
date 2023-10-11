import { Fragment } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { ChevronDownIcon } from '@heroicons/react/20/solid';
import dayjs from 'dayjs';
import { useQuery } from 'urql';

import {
  getMinimumPlanForLogRetention,
  transformPlans,
  type TimeRange,
} from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { graphql } from '@/gql';

const currentTime = new Date();

const timeRanges: TimeRange[] = [
  {
    start: new Date(currentTime.valueOf() - 30 * 60 * 1_000), // 30 Minutes
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 60 * 60 * 1_000), // 60 Minutes
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 6 * 60 * 60 * 1_000), // 6 Hours
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 12 * 60 * 60 * 1_000), // 12 Hours
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 24 * 60 * 60 * 1_000), // 24 Hours
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 3 * 24 * 60 * 60 * 1_000), // 3 Days
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 7 * 24 * 60 * 60 * 1_000), // 7 Days
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 14 * 24 * 60 * 60 * 1_000), // 14 Days
    end: currentTime,
  },
  {
    start: new Date(currentTime.valueOf() - 30 * 24 * 60 * 60 * 1_000), // 30 Days
    end: currentTime,
  },
];

export const defaultTimeRange = timeRanges[4]!;

const GetBillingPlanDocument = graphql(`
  query GetBillingPlan {
    account {
      plan {
        id
        name
        features
      }
    }

    plans {
      name
      features
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

  // Since "features" is a map, we can't be 100% sure that there's a log
  // retention value. So default to 7 days.
  let logRetention = 7;
  if (typeof data?.account.plan?.features?.log_retention === 'number') {
    logRetention = data?.account.plan?.features?.log_retention;
  }

  let plans: ReturnType<typeof transformPlans> | undefined;
  if (data?.plans) {
    plans = transformPlans(data?.plans);
  }

  return (
    <Listbox value={selectedTimeRange} onChange={onTimeRangeChange}>
      {({ open }) => (
        <>
          <Listbox.Label className="sr-only">Filter by time</Listbox.Label>
          <div className="relative">
            <Listbox.Button className="shadow-outline-secondary-light group inline-flex items-center gap-1 rounded-[6px] bg-slate-50 px-3 py-[5px] text-sm font-medium capitalize text-slate-800 hover:bg-slate-100 focus:outline-indigo-500">
              <p>Last {selectedTimeRange ? getTimeRangeLabel(selectedTimeRange) : '...'}</p>
              <ChevronDownIcon className="h-4 w-4" aria-hidden="true" />
            </Listbox.Button>

            <Transition
              show={open}
              as={Fragment}
              leave="transition ease-in duration-100"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Listbox.Options className="shadow-floating absolute left-0 z-10 mt-[5px] w-52 origin-top-left overflow-hidden rounded-md bg-white/95 py-[9px] ring-1 ring-black/5 backdrop-blur-[3px] focus:outline-none">
                {timeRanges.map((timeRange) => {
                  const timeRangeStartInDaysAgo = dayjs(currentTime).diff(
                    dayjs(timeRange.start),
                    'days'
                  );

                  const isPlanSufficient = timeRangeStartInDaysAgo <= logRetention;
                  let minimumPlanName: string | undefined = undefined;
                  if (plans) {
                    minimumPlanName = getMinimumPlanForLogRetention(plans, timeRangeStartInDaysAgo);
                  }

                  return (
                    <Listbox.Option
                      key={timeRange.end.valueOf() - timeRange.start.valueOf()}
                      className="ui-selected:bg-indigo-100 ui-disabled:text-slate-400 ui-selected:text-indigo-700 flex cursor-pointer select-none items-center justify-between px-3.5 py-1 text-sm font-medium capitalize text-slate-800 hover:bg-slate-100 focus:outline-none"
                      value={timeRange}
                      disabled={!isPlanSufficient}
                    >
                      {getTimeRangeLabel(timeRange)}{' '}
                      {!isPlanSufficient && minimumPlanName && (
                        <span className="inline-flex items-center rounded-sm px-[5px] py-0.5 text-[12px] font-semibold leading-tight text-indigo-500 ring-1 ring-inset ring-indigo-300">
                          {minimumPlanName} Plan
                        </span>
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
