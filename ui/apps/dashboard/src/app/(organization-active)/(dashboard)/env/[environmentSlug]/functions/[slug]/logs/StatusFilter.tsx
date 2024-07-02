'use client';

import { Fragment, useRef } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { getStatusBackgroundClass, getStatusBorderClass } from '@inngest/components/statusClasses';
import { RiArrowDownSLine, RiCloseLine } from '@remixicon/react';
import { noCase } from 'change-case';
import { titleCase } from 'title-case';

import { FunctionRunStatus } from '@/gql/graphql';
import cn from '@/utils/cn';
import getOrderedEnumValues from '@/utils/getOrderedEnumValues';

const orderedStatuses = getOrderedEnumValues(FunctionRunStatus, [
  FunctionRunStatus.Queued,
  FunctionRunStatus.Running,
  FunctionRunStatus.Paused,
  FunctionRunStatus.Cancelled,
  FunctionRunStatus.Completed,
  FunctionRunStatus.Failed,
]);

type StatusFilterProps = {
  selectedStatuses: FunctionRunStatus[];
  onStatusesChange: (statuses: FunctionRunStatus[]) => void;
  functionIsPaused?: boolean;
};

export default function StatusFilter({
  selectedStatuses,
  onStatusesChange,
  functionIsPaused,
}: StatusFilterProps) {
  const buttonRef = useRef<HTMLButtonElement>(null);
  function resetSelection(): void {
    onStatusesChange([]);
    // This is a hack to close the select menu since the Listbox component doesn't expose a way
    // to do this.
    buttonRef.current?.click();
  }

  const availableStatuses = orderedStatuses.filter((status) => {
    if (status === FunctionRunStatus.Paused) {
      return !!functionIsPaused;
    } else if (status === FunctionRunStatus.Running) {
      return !functionIsPaused;
    }
    return true;
  });

  const statusDots = availableStatuses.map((status) => {
    const isSelected = selectedStatuses.includes(status);
    return (
      <span
        key={status}
        className={cn(
          'inline-block h-[9px] w-[9px] flex-shrink-0 rounded-full border border-slate-50 bg-slate-50 ring-1 ring-inset ring-slate-300 group-hover:border-slate-100 [&:not(:first-child)]:-ml-1',
          isSelected && [getStatusBackgroundClass(status), getStatusBorderClass(status), 'ring-0']
        )}
        aria-hidden="true"
      />
    );
  });

  return (
    <Listbox value={selectedStatuses} onChange={onStatusesChange} multiple>
      {({ open }) => (
        <>
          <Listbox.Label className="sr-only">Filter by status</Listbox.Label>
          <div className="relative">
            <Listbox.Button
              ref={buttonRef}
              className="shadow-outline-secondary-light group inline-flex items-center gap-1 rounded-[6px] bg-slate-50 px-3 py-[5px] text-sm font-medium text-slate-800 hover:bg-slate-100 focus:outline-indigo-500"
            >
              <p>Status</p>
              <p className="sr-only">Filter by status</p>
              <span>{statusDots}</span>
              <RiArrowDownSLine className="h-4 w-4" aria-hidden="true" />
            </Listbox.Button>

            <Transition
              show={open}
              as={Fragment}
              leave="transition ease-in duration-100"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Listbox.Options className="shadow-floating absolute left-0 z-10 mt-[5px] w-52 origin-top-left overflow-hidden rounded-md bg-white/95 text-sm font-medium text-slate-800 ring-1 ring-black/5 backdrop-blur-[3px] focus:outline-none">
                <div className="py-[9px]">
                  {availableStatuses.map((status) => (
                    <Listbox.Option
                      key={status}
                      className="ui-active:bg-slate-100 flex cursor-pointer select-none items-center justify-between px-3.5 py-[5px] focus:outline-none"
                      value={status}
                    >
                      {({ selected }) => (
                        <>
                          <span className="inline-flex items-center gap-2">
                            <span
                              className={cn(
                                'inline-block h-[9px] w-[9px] flex-shrink-0 rounded-full',
                                [getStatusBackgroundClass(status), getStatusBorderClass(status)]
                              )}
                              aria-hidden="true"
                            />
                            <p>{titleCase(noCase(status))}</p>
                          </span>
                          <input
                            type="checkbox"
                            id={status}
                            checked={selected}
                            readOnly
                            className="border-muted h-[15px] w-[15px] rounded text-indigo-500 drop-shadow-sm checked:border-indigo-500 checked:drop-shadow-none"
                          />
                        </>
                      )}
                    </Listbox.Option>
                  ))}
                </div>
                {selectedStatuses.length > 0 ? (
                  <button
                    type="button"
                    className="inline-flex w-full items-center gap-1 border-t border-slate-100 p-2.5 text-[13px] font-semibold text-slate-500 hover:text-slate-700"
                    onClick={resetSelection}
                  >
                    <RiCloseLine className="h-[17px] w-[17px]" aria-hidden="true" />
                    <p>Reset Selection</p>
                  </button>
                ) : null}
              </Listbox.Options>
            </Transition>
          </div>
        </>
      )}
    </Listbox>
  );
}
