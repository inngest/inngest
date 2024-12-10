'use client';

import { Fragment } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { Pill } from '@inngest/components/Pill/Pill';
import { RiArrowDownSLine } from '@remixicon/react';
import { useQuery } from 'urql';

import GroupButton from '@/components/GroupButton/GroupButton';
import { graphql } from '@/gql';
import { FunctionRunTimeField } from '@/gql/graphql';

// export type TimeField = 'startedAt' | 'endedAt';
export const defaultTimeField = FunctionRunTimeField.StartedAt;

const fieldOptions = [
  { name: 'Queued', id: FunctionRunTimeField.StartedAt },
  { name: 'Ended', id: FunctionRunTimeField.EndedAt },
] as const satisfies Readonly<{ name: string; id: FunctionRunTimeField }[]>;

export type TimeRange = {
  start: Date;
  end: Date;
  key?: string;
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

const GetAccountEntitlementsDocument = graphql(`
  query GetAccountEntitlements {
    account {
      entitlements {
        history {
          limit
        }
      }
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
    query: GetAccountEntitlementsDocument,
  });

  const logRetention = data?.account.entitlements.history.limit || 7;

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
              <RiArrowDownSLine className="h-4 w-4" aria-hidden="true" />
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
                  className="mx-2 grid grid-flow-col justify-stretch"
                  handleClick={onTimeFieldChange}
                  options={fieldOptions}
                  selectedOption={selectedTimeField}
                  title="Select the time field to filter"
                />

                {timeRangeOptions.map((timeRange) => {
                  const isPlanSufficient = timeRange.daysAgo <= logRetention;
                  const label = getTimeRangeLabel(timeRange);

                  return (
                    <Listbox.Option
                      key={label}
                      className="ui-selected:bg-indigo-100 ui-disabled:text-slate-400 ui-selected:text-indigo-700 flex cursor-pointer select-none items-center justify-between px-3.5 py-1 text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-none"
                      value={timeRange.value}
                      disabled={!isPlanSufficient}
                    >
                      {label}{' '}
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
