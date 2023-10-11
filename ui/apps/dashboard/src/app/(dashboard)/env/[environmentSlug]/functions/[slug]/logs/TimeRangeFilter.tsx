'use client';

import { Fragment } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { ChevronDownIcon } from '@heroicons/react/20/solid';
import { useQuery } from 'urql';

import GroupButton from '@/components/GroupButton/GroupButton';
import { graphql } from '@/gql';
import { FunctionRunTimeField, type GetBillingPlanQuery } from '@/gql/graphql';

// export type TimeField = 'startedAt' | 'endedAt';
export const defaultTimeField = FunctionRunTimeField.StartedAt;

const fieldOptions = [
  { name: 'Started', id: FunctionRunTimeField.StartedAt },
  { name: 'Ended', id: FunctionRunTimeField.EndedAt },
] as const satisfies Readonly<{ name: string; id: FunctionRunTimeField }[]>;

export type TimeRange = {
  start: Date;
  end: Date;
};

type TimeRangeOption = {
  daysAgo: number;
  value: TimeRange;
};

const currentTime = new Date();

const timeRangeOptions: TimeRangeOption[] = [
  {
    daysAgo: 1,
    value: {
      start: new Date(currentTime.valueOf() - 24 * 60 * 60 * 1_000),
      end: currentTime,
    },
  },
  {
    daysAgo: 3,
    value: {
      start: new Date(currentTime.valueOf() - 3 * 24 * 60 * 60 * 1_000),
      end: currentTime,
    },
  },
  {
    daysAgo: 7,
    value: {
      start: new Date(currentTime.valueOf() - 7 * 24 * 60 * 60 * 1_000),
      end: currentTime,
    },
  },
  {
    daysAgo: 14,
    value: {
      start: new Date(currentTime.valueOf() - 14 * 24 * 60 * 60 * 1_000),
      end: currentTime,
    },
  },
  {
    daysAgo: 30,
    value: {
      start: new Date(currentTime.valueOf() - 30 * 24 * 60 * 60 * 1_000),
      end: currentTime,
    },
  },
];

export const defaultTimeRange = timeRangeOptions[1]!.value;

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

type TimeRangeFilterProps = {
  selectedTimeField: FunctionRunTimeField;
  selectedTimeRange: TimeRange;
  onTimeFieldChange: (timeField: FunctionRunTimeField) => void;
  onTimeRangeChange: (timeRange: TimeRange) => void;
};

export default function TimeRangeFilter({
  selectedTimeField,
  selectedTimeRange,
  onTimeFieldChange,
  onTimeRangeChange,
}: TimeRangeFilterProps) {
  const [{ data }] = useQuery({
    query: GetBillingPlanDocument,
  });

  // Since "features" is a map, we can't be 100% sure that there's a log
  // retention value. So default to 7 days.
  let logRetention = 7;
  if (typeof data?.account.plan?.features?.log_retention === 'number') {
    logRetention = data?.account.plan?.features?.log_retention;
  }

  let plans: Plan[] | undefined;
  if (data?.plans) {
    plans = transformPlans(data?.plans);
  }

  const selectedTimeRangeOption = timeRangeOptions.find(
    (option) => option.value === selectedTimeRange
  );

  return (
    <Listbox value={selectedTimeRange} onChange={onTimeRangeChange}>
      {({ open }) => (
        <>
          <Listbox.Label className="sr-only">Filter by time</Listbox.Label>
          <div className="relative">
            <Listbox.Button className="shadow-outline-secondary-light group inline-flex items-center gap-1 rounded-[6px] bg-slate-50 px-3 py-[5px] text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-indigo-500">
              <p>
                {getTimeFieldLabel(selectedTimeField)} in Last{' '}
                {selectedTimeRangeOption ? getTimeRangeLabel(selectedTimeRangeOption) : ''}
              </p>
              <ChevronDownIcon className="h-4 w-4" aria-hidden="true" />
            </Listbox.Button>

            <Transition
              show={open}
              as={Fragment}
              leave="transition ease-in duration-100"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Listbox.Options className="shadow-floating absolute left-0 z-10 mt-[5px] w-52 origin-top-left overflow-hidden rounded-md  bg-white/95 py-[9px] ring-1 ring-black/5 backdrop-blur-[3px] focus:outline-none">
                <GroupButton
                  className="justify-stretch mx-2 grid grid-flow-col"
                  handleClick={onTimeFieldChange}
                  options={fieldOptions}
                  selectedOption={selectedTimeField}
                  title="Select the time field to filter"
                />

                {timeRangeOptions.map((timeRange, index) => {
                  const isPlanSufficient = timeRange.daysAgo <= logRetention;
                  const label = getTimeRangeLabel(timeRange);

                  let minimumPlanName: string | undefined = undefined;
                  if (plans) {
                    minimumPlanName = getMinimumPlanForLogRetention(plans, timeRange.daysAgo);
                  }

                  return (
                    <Listbox.Option
                      key={label}
                      className="ui-selected:bg-indigo-100 ui-disabled:text-slate-400 ui-selected:text-indigo-700 flex cursor-pointer select-none items-center justify-between px-3.5 py-1 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none"
                      value={timeRange.value}
                      disabled={!isPlanSufficient}
                    >
                      {label}{' '}
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

function getTimeFieldLabel(timeField: FunctionRunTimeField): string {
  const match = fieldOptions.find((option) => option.id === timeField);

  // Should be impossible.
  if (!match) {
    throw new Error('invalid time field');
  }

  return match.name;
}

function getTimeRangeLabel(timeRangeOption: TimeRangeOption): string {
  if (timeRangeOption.daysAgo === 1) {
    return '24 Hours';
  }

  return `${timeRangeOption.daysAgo} Days`;
}

export function getMinimumPlanForLogRetention(
  plans: Plan[],
  logRetention: number
): string | undefined {
  // Sort plans by ascending log retention. This is needed because we'll need to
  // find the "lowest" plan that supports the specified log retention.
  plans = [...plans].sort((a, b) => {
    return a.logRetention - b.logRetention;
  });

  for (const plan of plans) {
    if (plan.logRetention >= logRetention) {
      return plan.name;
    }
  }

  // TODO: This probably shouldn't be hardcoded.
  return 'Enterprise';
}

type Plan = {
  name: string;
  logRetention: number;
};

export function transformPlans(plans: GetBillingPlanQuery['plans']): Plan[] {
  const newPlans: Plan[] = [];

  for (const plan of plans) {
    if (!plan || typeof plan.features?.log_retention !== 'number') {
      continue;
    }

    newPlans.push({
      name: plan.name,
      logRetention: plan.features.log_retention,
    });
  }

  return newPlans;
}
