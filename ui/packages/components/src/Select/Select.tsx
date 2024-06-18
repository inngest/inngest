import { Listbox } from '@headlessui/react';
import { RiArrowDownSLine } from '@remixicon/react';

import { Checkbox } from '../Checkbox';
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
          isLabelVisible && 'divide-muted bg-canvasSubtle divide-x',
          'border-muted flex items-center rounded-md border text-sm',
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
}: React.PropsWithChildren<{ isLabelVisible?: boolean }>) {
  return (
    <Listbox.Button
      className={cn(
        !isLabelVisible && 'rounded-l-[5px]',
        'bg-surfaceBase flex h-10 w-full items-center justify-between rounded-r-[5px] px-2'
      )}
    >
      {children}
      <RiArrowDownSLine
        className="ui-open:-rotate-180 text-subtle h-4 w-4 transition-transform duration-500"
        aria-hidden="true"
      />
    </Listbox.Button>
  );
}

function Options({ children }: React.PropsWithChildren) {
  return (
    <Listbox.Options className="absolute mt-1 min-w-max">
      <div className="border-muted bg-surfaceBase overflow-hidden rounded-md border py-1 drop-shadow-lg">
        {children}
      </div>
    </Listbox.Options>
  );
}

function Option({ children, option }: React.PropsWithChildren<{ option: Option }>) {
  return (
    <Listbox.Option
      className=" ui-selected:text-success ui-selected:font-medium ui-active:bg-canvasSubtle/50 flex select-none items-center justify-between focus:outline-none"
      key={option.id}
      value={option}
      disabled={option.disabled}
    >
      <div className="ui-selected:border-success my-2 w-full border-l-2 border-transparent pl-5 pr-4">
        {children}
      </div>
    </Listbox.Option>
  );
}

function CheckboxOption({ children, option }: React.PropsWithChildren<{ option: Option }>) {
  return (
    <Listbox.Option
      className=" ui-active:bg-canvasSubtle/50 flex select-none items-center justify-between py-1.5 pl-2 pr-4 focus:outline-none"
      key={option.id}
      value={option}
    >
      {({ selected }: { selected: boolean }) => (
        <span className="inline-flex items-center">
          <span className="inline-flex items-center gap-2">
            <Checkbox
              id={option.id}
              checked={selected}
              disabled={option.disabled}
              className="h-4 w-4"
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
