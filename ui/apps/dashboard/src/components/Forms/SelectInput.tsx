import { useEffect, useState } from 'react';
import * as Select from '@radix-ui/react-select';
import { RiArrowDownSLine, RiCheckLine } from '@remixicon/react';

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

/**
 * @deprecated Use shared Select component instead
 */
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
      <Select.Trigger className="border-muted bg-canvasBase outline-primary-moderate data-[placeholder]:text-light flex items-center justify-between rounded-md border px-3 py-1.5 text-sm leading-none outline-2 outline-offset-2 transition-all focus:outline">
        <Select.Value placeholder={props.placeholder} />
        <Select.Icon className="">
          <RiArrowDownSLine className="h-5" />
        </Select.Icon>
      </Select.Trigger>

      <Select.Content
        className="border-muted bg-canvasBase outline-primary-moderate z-10 w-[var(--radix-select-trigger-width)] rounded-md border py-1 text-sm leading-none shadow outline-2 outline-offset-2 transition-all focus:outline"
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
                className="hover:bg-canvasSubtle hover:text-primary-moderate data-[disabled]:text-disabled flex cursor-pointer flex-col gap-2 px-3 py-2 font-medium outline-none hover:outline-none data-[disabled]:cursor-not-allowed"
              >
                <span className="flex flex-row items-center">
                  <Select.ItemText>{opt.label}</Select.ItemText>
                  <Select.ItemIndicator>
                    <RiCheckLine className="text-primary-moderate ml-2 h-4" />
                  </Select.ItemIndicator>
                </span>
                {opt.description && (
                  <span className="text-subtle block text-xs font-normal">{opt.description}</span>
                )}
              </Select.Item>
            );
          })}
        </Select.Viewport>
      </Select.Content>
    </Select.Root>
  );
}
