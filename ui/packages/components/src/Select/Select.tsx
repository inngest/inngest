import { Listbox } from '@headlessui/react';
import { RiArrowDownSLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

type SelectProps = {
  label?: string;
  defaultValue?: string[];
  onChange: (value: string[]) => void;
  isLabelVisible?: boolean;
  children: React.ReactNode;
};

export function Select({
  defaultValue,
  label,
  isLabelVisible = true,
  children,
  onChange,
}: SelectProps) {
  return (
    <Listbox value={defaultValue} onChange={onChange} multiple={multiple}>
      <span
        className={cn(
          isLabelVisible && 'divide-x divide-slate-300',
          'flex items-center rounded-md border border-slate-300 bg-slate-50 text-sm'
        )}
      >
        <Listbox.Label
          className={cn(!isLabelVisible && 'sr-only', 'rounded-l-[5px] px-2 py-2.5 text-slate-600')}
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
        'flex items-center rounded-r-[5px] bg-white px-2 py-3'
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
      className="ui-active:bg-blue-50 flex select-none items-center justify-between px-2 py-4 focus:outline-none"
      key={option}
      value={option}
    >
      {children}
    </Listbox.Option>
  );
}

Select.Button = Button;
Select.Options = Options;
Select.Option = Option;
Select.CustomOption = Listbox.Option;
