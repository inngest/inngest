import { forwardRef } from 'react';
import { Combobox, Listbox, type ComboboxInputProps } from '@headlessui/react';
import { RiArrowDownSLine } from '@remixicon/react';

import { Button as InngestButton } from '../Button';
import { Checkbox } from '../Checkbox';
import { cn } from '../utils/classNames';

type SelectProps = {
  label?: string;
  isLabelVisible?: boolean;
  children: React.ReactNode;
  className?: string;
  size?: 'small' | 'medium';
};

export type Option = {
  id: string;
  name: string;
  disabled?: boolean;
};

type MultiProps = {
  onChange: (value: Option[]) => void;
  value?: Option[];
  multiple: true;
};

type SingleProps = {
  onChange: (value: Option) => void;
  value?: Option | null;
  multiple?: false;
};

type Props = SelectProps & (MultiProps | SingleProps);

export function Select({
  value,
  label,
  isLabelVisible = true,
  children,
  onChange,
  multiple,
  className,
}: Props) {
  return (
    <Listbox value={value} onChange={onChange} multiple={multiple}>
      <span
        className={cn(
          isLabelVisible && 'divide-muted bg-canvasSubtle text-basis divide-x',
          'border-muted flex items-center rounded-md border text-sm',
          className
        )}
      >
        <Listbox.Label
          className={cn(!isLabelVisible && 'sr-only', 'rounded-l-[5px] px-2 capitalize')}
        >
          {label}
        </Listbox.Label>
        <span className="relative w-full">{children}</span>
      </span>
    </Listbox>
  );
}

type ButtonProps = {
  children: React.ReactNode;
  isLabelVisible?: boolean;
  className?: string;
  as: React.ElementType;
  size?: 'small' | 'medium';
};

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ children, isLabelVisible, className, as: Component, size = 'medium' }, ref) => {
    return (
      <Component
        ref={ref}
        className={cn(
          !isLabelVisible && 'rounded-l-[5px]',
          size === 'small' ? 'h-[30px]' : 'h-[38px]',
          'bg-surfaceBase text-basis flex w-full items-center justify-between rounded-r-[5px] px-2',
          className
        )}
      >
        {children}
        <RiArrowDownSLine
          className="ui-open:-rotate-180 text-muted h-4 w-4 transition-transform duration-500"
          aria-hidden="true"
        />
      </Component>
    );
  }
);
Button.displayName = 'Button';

function Options({
  children,
  as: Component,
  className,
}: React.PropsWithChildren<{ as: React.ElementType; className?: string }>) {
  return (
    <Component className={cn('absolute z-10 mt-1 min-w-max', className)}>
      <div className="border-muted bg-surfaceBase shadow-primary z-10 overflow-hidden rounded-md border py-1">
        {children}
      </div>
    </Component>
  );
}

function Option({
  children,
  option,
  as: Component,
}: React.PropsWithChildren<{ option: Option; as: React.ElementType }>) {
  return (
    <Component
      className=" ui-selected:text-success ui-selected:font-medium ui-active:bg-canvasSubtle/50 text-basis flex select-none items-center justify-between focus:outline-none"
      key={option.id}
      value={option}
      disabled={option.disabled}
    >
      <div className="ui-selected:border-success my-2 w-full border-l-2 border-transparent pl-5 pr-4">
        {children}
      </div>
    </Component>
  );
}

function CheckboxOption({
  children,
  option,
  as: Component,
}: React.PropsWithChildren<{ option: Option; as: React.ElementType }>) {
  return (
    <Component
      className=" ui-active:bg-canvasSubtle/50 text-basis flex select-none items-center justify-between py-1.5 pl-2 pr-4 focus:outline-none"
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
    </Component>
  );
}

function Footer({
  onReset,
  onApply,
  disabledReset,
  disabledApply,
}: {
  onReset?: () => void;
  onApply?: () => void;
  disabledReset?: boolean;
  disabledApply?: boolean;
}) {
  return (
    <div
      className={cn(
        'border-muted mt-1 flex items-center border-t px-2 pb-1 pt-2',
        onReset ? 'justify-between' : 'justify-end'
      )}
    >
      {onReset && (
        <InngestButton
          label="Reset"
          appearance="ghost"
          size="small"
          onClick={onReset}
          disabled={disabledReset}
        />
      )}
      {onApply && (
        <InngestButton label="Apply" size="small" onClick={onApply} disabled={disabledApply} />
      )}
    </div>
  );
}

Select.Button = forwardRef<HTMLButtonElement, Omit<ButtonProps, 'as'>>((props, ref) => (
  <Button {...props} ref={ref} as={Listbox.Button} />
));
Select.Options = (props: React.PropsWithChildren<{ className?: string }>) => (
  <Options {...props} as={Listbox.Options} />
);
Select.Option = (props: React.PropsWithChildren<{ option: Option }>) => (
  <Option {...props} as={Listbox.Option} />
);
Select.CheckboxOption = (props: React.PropsWithChildren<{ option: Option }>) => (
  <CheckboxOption {...props} as={Listbox.Option} />
);
Select.Footer = Footer;

export function SelectWithSearch({
  value,
  label,
  isLabelVisible = true,
  children,
  onChange,
  multiple,
  className,
}: Props) {
  const content = (
    <span
      className={cn(
        isLabelVisible && 'divide-muted bg-canvasSubtle text-basis divide-x',
        'border-muted flex items-center rounded-md border text-sm',
        className
      )}
    >
      <Combobox.Label
        className={cn(!isLabelVisible && 'sr-only', 'rounded-l-[5px] px-2 capitalize')}
      >
        {label}
      </Combobox.Label>
      <span className="relative w-full">{children}</span>
    </span>
  );

  // This conditional is only necessary because of a TypeScript limitation: it's
  // having a hard time understanding how the types of `value` and
  // `onChange` vary with `multiple`
  if (multiple) {
    return (
      <Combobox value={value} onChange={onChange} multiple={multiple}>
        {content}
      </Combobox>
    );
  }

  return (
    <Combobox value={value} onChange={onChange} multiple={multiple}>
      {content}
    </Combobox>
  );
}

function Search<T>({ ...props }: ComboboxInputProps<'input', T>) {
  return (
    <div className="mx-2 my-2">
      <Combobox.Input
        className="border-subtle text-basis bg-surfaceBase placeholder:text-disabled focus-visible:outline-primary-moderate w-full rounded-md border px-4 py-2 text-sm"
        {...props}
      />
    </div>
  );
}

SelectWithSearch.Button = forwardRef<HTMLButtonElement, Omit<ButtonProps, 'as'>>((props, ref) => (
  <Button {...props} ref={ref} as={Combobox.Button} />
));
SelectWithSearch.Options = (props: React.PropsWithChildren<{ className?: string }>) => (
  <Options {...props} as={Combobox.Options} />
);
SelectWithSearch.Option = (props: React.PropsWithChildren<{ option: Option }>) => (
  <Option {...props} as={Combobox.Option} />
);
SelectWithSearch.CheckboxOption = (props: React.PropsWithChildren<{ option: Option }>) => (
  <CheckboxOption {...props} as={Combobox.Option} />
);
SelectWithSearch.SearchInput = Search;
SelectWithSearch.Footer = Footer;

// Used as a wrapper when we group select components in something similar to a button group
export function SelectGroup({ children }: React.PropsWithChildren) {
  return (
    <div className="flex items-center [&>*:first-child]:rounded-r-none [&>*:last-child]:rounded-l-none [&>*:not(:first-child)]:border-l-0">
      {children}
    </div>
  );
}
