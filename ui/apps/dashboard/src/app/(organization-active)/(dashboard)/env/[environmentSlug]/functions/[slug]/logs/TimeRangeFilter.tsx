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
            <div className=" border-muted bg-surfaceBase flex items-center rounded-md border text-sm">
              <Listbox.Button className="text-basis group flex h-[38px] w-full items-center justify-between rounded-r-[5px] px-2">
                <p>
                  {getTimeFieldLabel(selectedTimeField)} in Last{' '}
                  {selectedTimeRangeOption ? getTimeRangeLabel(selectedTimeRangeOption) : ''}
                </p>
                <RiArrowDownSLine
                  className="ui-open:-rotate-180 text-muted h-4 w-4 transition-transform duration-500"
                  aria-hidden="true"
                />
              </Listbox.Button>
            </div>

            <Transition
              show={open}
              as={Fragment}
              leave="transition ease-in duration-100"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Listbox.Options className="border-muted bg-surfaceBase absolute left-0 z-10 mt-[5px] w-52 origin-top-left overflow-hidden rounded-md border py-1 drop-shadow-lg focus:outline-none">
                <GroupButton
                  className="mx-2 mb-1 grid grid-flow-col justify-stretch"
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
                      className="ui-selected:bg-canvasSubtle ui-disabled:text-disabled ui-selected:text-primary-moderate text-basis ui-active:bg-canvasSubtle/50 flex cursor-pointer select-none items-center justify-between px-3.5 py-1 text-sm focus:outline-none"
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
