import { Listbox } from '@headlessui/react';
import ChevronDownIcon from '@heroicons/react/20/solid/ChevronDownIcon';

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
    <Listbox value={defaultValue} onChange={onChange} multiple>
      <>
        <span
          className={cn(
            isLabelVisible && 'divide-x divide-slate-300',
            'flex rounded-md border border-slate-300 text-sm'
          )}
        >
          <Listbox.Label
            className={cn(
              !isLabelVisible && 'sr-only',
              'rounded-l-[5px] bg-slate-50 px-2 py-2.5 text-slate-600'
            )}
          >
            {label}
          </Listbox.Label>
          <span className="relative">{children}</span>
        </span>
      </>
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
      <ChevronDownIcon className="h-4 w-4 text-slate-500" aria-hidden="true" />
    </Listbox.Button>
  );
}

type OptionProps = React.PropsWithChildren<{
  children: (option: string) => React.ReactNode;
  options: string[];
  multiple?: boolean;
}>;

function Options({ children, options, multiple }: React.PropsWithChildren<OptionProps>) {
  return (
    <Listbox.Options className="absolute mt-1 min-w-max">
      <div className="overflow-hidden rounded-md border border-slate-200 bg-white drop-shadow-lg">
        {options.map((option) => {
          return (
            <Listbox.Option
              key={option}
              className="ui-active:bg-blue-50 flex select-none items-center justify-between px-2 py-4 focus:outline-none"
              value={option}
            >
              {({ selected }) => (
                <>
                  <span className="inline-flex items-center gap-2 lowercase">
                    {multiple && (
                      <CheckList selected={selected} option={option}>
                        {children}
                      </CheckList>
                    )}
                    {!multiple && <p className="text-sm first-letter:capitalize">{option}</p>}
                  </span>
                </>
              )}
            </Listbox.Option>
          );
        })}
      </div>
    </Listbox.Options>
  );
}

Select.Button = Button;
Select.Options = Options;

type Cenas = {
  selected: boolean;
  option: string;
  children: (option: string) => React.ReactNode;
};

function CheckList({ selected, option, children }: Cenas) {
  return (
    <span className="inline-flex items-center gap-2 lowercase">
      <input
        type="checkbox"
        id={option}
        checked={selected}
        readOnly
        className="h-[15px] w-[15px] rounded border-slate-300 text-indigo-500 drop-shadow-sm checked:border-indigo-500 checked:drop-shadow-none"
      />
      <span className="flex items-center gap-1">
        {children(option)}
        <label className="text-sm first-letter:capitalize">{option}</label>
      </span>
    </span>
  );
}
