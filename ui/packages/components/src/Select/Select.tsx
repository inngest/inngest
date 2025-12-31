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
  size = 'medium',
}: Props) {
  return (
    <Listbox value={value} onChange={onChange} multiple={multiple}>
      {({ open }) => (
        <span
          className={cn(
            isLabelVisible && 'divide-muted text-muted divide-x',
            'disabled:bg-disabled disabled:text-disabled border-muted flex items-center rounded border',
            size === 'small' ? 'text-[13px]' : 'text-sm',
            className
          )}
        >
          <Listbox.Label
            className={cn(
              !isLabelVisible && 'sr-only',
              'rounded-l px-2 capitalize',
              size === 'small' ? 'text-xs' : 'text-sm'
            )}
          >
            {label}
          </Listbox.Label>
          <span className="relative w-full">{children}</span>
        </span>
      )}
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
          !isLabelVisible && 'rounded-l',
          // Real height is 26px and 34px, but we add the 2px border
          size === 'small' ? 'h-[24px] text-xs' : 'h-[32px] py-1.5 text-sm',
          'disabled:bg-disabled disabled:text-disabled text-basis placeholder:text-disabled flex w-full items-center justify-between gap-1 rounded-r px-1.5',
          className
        )}
      >
        {children}
        <RiArrowDownSLine
          className="ui-open:-rotate-180 text-subtle h-4 w-4 transition-transform duration-200"
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
      <div className="border-subtle bg-surfaceBase shadow-primary z-10 overflow-hidden rounded border py-1">
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
      className="ui-selected:text-secondary-intense ui-selected:font-medium ui-active:bg-canvasSubtle/50 text-basis ui-disabled:text-disabled ui-disabled:cursor-not-allowed flex select-none items-center justify-between px-4 py-1.5 focus:outline-none"
      key={option.id}
      value={option}
      disabled={option.disabled}
    >
      {children}
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
      className=" ui-active:bg-canvasSubtle/50 text-basis flex select-none items-center justify-between px-4 py-1.5 focus:outline-none"
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
        'border-subtle mt-1 flex items-center border-t px-2 pb-1 pt-2',
        onReset ? 'justify-between' : 'justify-end'
      )}
    >
      {onReset && (
        <InngestButton
          label="Reset"
          appearance="ghost"
          kind="secondary"
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
  size,
}: Props) {
  const renderContent = (open: boolean) => (
    <span
      className={cn(
        isLabelVisible && 'divide-muted text-muted divide-x',
        'disabled:bg-disabled disabled:text-disabled border-muted flex items-center rounded border',
        size === 'small' ? 'text-[13px]' : 'text-sm',
        className
      )}
    >
      <Combobox.Label
        className={cn(
          !isLabelVisible && 'sr-only',
          'rounded-l px-2 capitalize',
          size === 'small' ? 'text-xs' : 'text-sm'
        )}
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
        {({ open }) => renderContent(open)}
      </Combobox>
    );
  }

  return (
    <Combobox value={value} onChange={onChange} multiple={multiple}>
      {({ open }) => renderContent(open)}
    </Combobox>
  );
}

function Search<T>({ ...props }: ComboboxInputProps<'input', T>) {
  return (
    <div className="mx-3 my-2">
      <Combobox.Input
        className="border-subtle focus-visible:border-active text-basis bg-surfaceBase placeholder:text-disabled w-full rounded border px-2 py-1 text-sm outline-none focus-visible:outline-none focus-visible:ring-0"
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
