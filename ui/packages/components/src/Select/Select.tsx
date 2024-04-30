import { Listbox } from '@headlessui/react';
import { RiArrowDownSLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

type SelectProps = {
  label?: string;
  isLabelVisible?: boolean;
  children: React.ReactNode;
};

type MultiProps = {
  onChange: (value: string[]) => void;
  defaultValue?: string[];
  multiple: true;
};

type SingleProps = {
  onChange: (value: string) => void;
  defaultValue?: string;
  multiple?: false;
};

export function Select({
  defaultValue,
  label,
  isLabelVisible = true,
  children,
  onChange,
  multiple,
}: SelectProps & (MultiProps | SingleProps)) {
  return (
    <Listbox value={defaultValue} onChange={onChange} multiple={multiple}>
      <span
        className={cn(
          isLabelVisible && 'divide-x divide-slate-300',
          'flex items-center rounded-md border border-slate-300 bg-slate-50 text-sm'
        )}
      >
        <Listbox.Label
          className={cn(!isLabelVisible && 'sr-only', 'rounded-l-[5px] px-2 text-slate-600')}
        >
          {label}
        </Listbox.Label>
        <span className="relative">{children}</span>
      </span>
    </Listbox>
  );
}

function Button({
  children,
  isLabelVisible,
}: React.PropsWithChildren<{ isLabelVisible?: boolean }>) {
  return (
    <Listbox.Button
      className={cn(
        !isLabelVisible && 'rounded-l-[5px]',
        'flex h-10 items-center rounded-r-[5px] bg-white px-2'
      )}
    >
      {children}
      <RiArrowDownSLine className="h-4 w-4 text-slate-500" aria-hidden="true" />
    </Listbox.Button>
  );
}

function Options({ children }: React.PropsWithChildren) {
  return (
    <Listbox.Options className="absolute mt-1 min-w-max">
      <div className="overflow-hidden rounded-md border border-slate-200 bg-white py-1 drop-shadow-lg">
        {children}
      </div>
    </Listbox.Options>
  );
}

function Option({ children, option }: React.PropsWithChildren<{ option: string }>) {
  return (
    <Listbox.Option
      className=" ui-selected:text-indigo-500 ui-selected:font-medium ui-active:bg-blue-50 flex select-none items-center justify-between focus:outline-none"
      key={option}
      value={option}
    >
      <div className="ui-selected:border-indigo-500 my-2 border-l-2 border-transparent pl-5 pr-4">
        {children}
      </div>
    </Listbox.Option>
  );
}

function CheckboxOption({ children, option }: React.PropsWithChildren<{ option: string }>) {
  return (
    <Listbox.Option
      className=" ui-active:bg-blue-50 flex select-none items-center justify-between py-1.5 pl-2 pr-4 focus:outline-none"
      key={option}
      value={option}
    >
      {({ selected }: { selected: boolean }) => (
        <span className="inline-flex items-center">
          <span className="inline-flex items-center gap-2">
            <input
              type="checkbox"
              id={option}
              checked={selected}
              readOnly
              className="h-[15px] w-[15px] rounded border-slate-300 text-indigo-500 drop-shadow-sm checked:border-indigo-500 checked:drop-shadow-none"
            />
            {children}
          </span>
        </span>
      )}
    </Listbox.Option>
  );
}

Select.Button = Button;
Select.Options = Options;
Select.Option = Option;
Select.CheckboxOption = CheckboxOption;

// Used as a wrapper when we group select components in something similar to a button group
export function SelectGroup({ children }: React.PropsWithChildren) {
  return (
    <div className="flex items-center [&>*:first-child]:rounded-r-none [&>*:last-child]:rounded-l-none [&>*:not(:first-child)]:border-l-0">
      {children}
    </div>
  );
}
