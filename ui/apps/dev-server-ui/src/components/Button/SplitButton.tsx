import { useState } from 'react';
import * as Select from '@radix-ui/react-select';

import { IconChevron } from '@/icons';
import classNames from '@/utils/classnames';
import Button from './Button';
import { getButtonColors, getButtonSizeStyles, getDisabledStyles } from './buttonStyles';

type ButtonCopyProps = {
  kind?: 'default' | 'primary' | 'success' | 'danger';
  size?: 'small' | 'regular' | 'large';
  items: {
    label: string;
    icon?: React.ReactNode;
    onClick: () => void;
  }[];
};

export default function SplitButton({ kind = 'default', size = 'small', items }: ButtonCopyProps) {
  const [value, setValue] = useState(items.length > 0 ? items[0]?.label : '');
  const selectedItem = items.find((item) => item.label === value);
  const { onClick: btnAction, label, icon } = selectedItem || {};

  const buttonColors = getButtonColors({ kind, appearance: 'solid' });
  const buttonSizes = getButtonSizeStyles({ size, icon: true });
  const dropdownSizes = getButtonSizeStyles({ size, label: '' });
  const disabledStyles = getDisabledStyles();
  const verticalDivider =
    'before:absolute before:h-3/4 before:border-l before:border-slate-800/50 before:top-2/4 before:left-0 before:-translate-x-2/4 before:-translate-y-2/4';

  return (
    <div className="flex items-center">
      <Button
        btnAction={btnAction}
        label={label}
        isSplit
        kind={kind}
        size={size}
        appearance="solid"
        icon={icon}
      />
      <Select.Root value={value} onValueChange={setValue} name="Open for more actions">
        <Select.Trigger
          className={classNames(
            buttonColors,
            buttonSizes,
            disabledStyles,
            'relative flex items-center justify-center gap-1.5 rounded-r border-l-transparent drop-shadow-sm transition-all active:scale-95',
            verticalDivider
          )}
        >
          <Select.Icon>
            <IconChevron />
          </Select.Icon>
        </Select.Trigger>

        <Select.Portal>
          <Select.Content
            className="z-50 cursor-pointer overflow-hidden rounded bg-slate-800 text-white"
            position="popper"
            align="end"
            sideOffset={0}
          >
            <Select.ScrollUpButton />
            <Select.Viewport>
              {items.map((item, index) => (
                <Select.Item
                  className={classNames(
                    dropdownSizes,
                    'data-[highlighted]:bg-slate-500 data-[state=checked]:bg-slate-600 data-[state=checked]:data-[highlighted]:bg-slate-500'
                  )}
                  key={index}
                  value={item.label}
                >
                  <Select.ItemText>{item.label}</Select.ItemText>
                </Select.Item>
              ))}
            </Select.Viewport>
            <Select.ScrollDownButton />
            <Select.Arrow />
          </Select.Content>
        </Select.Portal>
      </Select.Root>
    </div>
  );
}
