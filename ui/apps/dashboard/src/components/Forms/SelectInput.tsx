import { useEffect, useState } from 'react';
import { CheckIcon, ChevronDownIcon } from '@heroicons/react/20/solid';
import * as Select from '@radix-ui/react-select';

export type SelectOption<T extends string> = {
  label: string | React.ReactNode;
  value: T;
  description?: string;
  disabled?: boolean;
};

export interface SelectProps<T extends string> {
  value: T | null;
  options: SelectOption<T>[];
  onChange: (value: T) => void;
  placeholder: string;
  required?: boolean;
}

export function SelectInput<T extends string>(props: SelectProps<T>) {
  // Key is to fix bug with radix-ui/react-select
  // https://github.com/radix-ui/primitives/issues/1569
  const [key, setKey] = useState(0);

  useEffect(() => {
    if (props.value === null) {
      setKey((k) => k + 1);
    }
  }, [props.value]);

  return (
    <Select.Root
      key={key}
      value={props.value || undefined}
      onValueChange={props.onChange}
      required={props.required}
    >
      <Select.Trigger className="flex items-center justify-between rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm leading-none shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline data-[placeholder]:text-slate-500">
        <Select.Value placeholder={props.placeholder} />
        <Select.Icon className="">
          <ChevronDownIcon className="h-5" />
        </Select.Icon>
      </Select.Trigger>

      <Select.Content
        className="w-[var(--radix-select-trigger-width)] rounded-lg border border-slate-300 bg-white py-1 text-sm leading-none shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline"
        position="popper"
        sideOffset={8}
      >
        <Select.Viewport>
          {props.options.map((opt) => {
            return (
              <Select.Item
                value={opt.value}
                key={opt.value}
                disabled={opt.disabled}
                className="flex cursor-pointer flex-col gap-2 px-3 py-2 font-medium outline-none hover:bg-slate-100 hover:text-indigo-500 hover:outline-none data-[disabled]:cursor-not-allowed data-[disabled]:text-slate-500 data-[disabled]:hover:bg-slate-50"
              >
                <span className="flex flex-row items-center">
                  <Select.ItemText>{opt.label}</Select.ItemText>
                  <Select.ItemIndicator>
                    <CheckIcon className="ml-2 h-4 text-indigo-500" />
                  </Select.ItemIndicator>
                </span>
                {opt.description && (
                  <span className="block text-xs font-normal text-slate-500">
                    {opt.description}
                  </span>
                )}
              </Select.Item>
            );
          })}
        </Select.Viewport>
      </Select.Content>
    </Select.Root>
  );
}
