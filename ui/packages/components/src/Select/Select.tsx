import { Listbox } from '@headlessui/react';
import { RiArrowDownSLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

type SelectProps = {
  label?: string;
  isLabelVisible?: boolean;
  children: React.ReactNode;
  className?: string;
};

export type Option = {
  id: string;
  name: string;
  disabled?: boolean;
};

type MultiProps = {
  onChange: (value: Option[]) => void;
  defaultValue?: Option[];
  multiple: true;
};

type SingleProps = {
  onChange: (value: Option) => void;
  defaultValue?: Option;
  multiple?: false;
};

type Props = SelectProps & (MultiProps | SingleProps);

export function Select({
  defaultValue,
  label,
  isLabelVisible = true,
  children,
  onChange,
  multiple,
  className,
}: Props) {
  return (
    <Listbox value={defaultValue} onChange={onChange} multiple={multiple}>
      <span
        className={cn(
          isLabelVisible && 'divide-x divide-slate-300',
          'flex items-center rounded-md border border-slate-300 bg-slate-50 text-sm text-slate-600',
          className
        )}
      >
        <Listbox.Label className={cn(!isLabelVisible && 'sr-only', 'rounded-l-[5px] px-2')}>
          {label}
        </Listbox.Label>
        <span className="relative w-full">{children}</span>
      </span>
    </Listbox>
  );
}

function Button({
  children,
  isLabelVisible,
  className,
}: React.PropsWithChildren<{ isLabelVisible?: boolean }> & { className?: string }) {
  return (
    <Listbox.Button
      className={cn(
        !isLabelVisible && 'rounded-l-[5px]',
        'flex h-10 w-full items-center justify-between rounded-r-[5px] bg-white px-2',
        className
      )}
    >
      {children}
      <RiArrowDownSLine
        className="ui-open:-rotate-180 h-4 w-4 text-slate-500 transition-transform duration-500"
        aria-hidden="true"
      />
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

function Option({ children, option }: React.PropsWithChildren<{ option: Option }>) {
  return (
    <Listbox.Option
      className=" ui-selected:text-indigo-500 ui-selected:font-medium ui-active:bg-blue-50 flex select-none items-center justify-between focus:outline-none"
      key={option.id}
      value={option}
      disabled={option.disabled}
    >
      <div className="ui-selected:border-indigo-500 my-2 w-full border-l-2 border-transparent pl-5 pr-4">
        {children}
      </div>
    </Listbox.Option>
  );
}

function CheckboxOption({ children, option }: React.PropsWithChildren<{ option: Option }>) {
  return (
    <Listbox.Option
      className=" ui-active:bg-blue-50 flex select-none items-center justify-between py-1.5 pl-2 pr-4 focus:outline-none"
      key={option.id}
      value={option}
    >
      {({ selected }: { selected: boolean }) => (
        <span className="inline-flex items-center">
          <span className="inline-flex items-center gap-2">
            <input
              type="checkbox"
              id={option.id}
              checked={selected}
              disabled={option.disabled}
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
