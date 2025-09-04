'use client';

import { forwardRef } from 'react';
import { cn } from '@inngest/components/utils/classNames';

import { Tab, type TabProps } from './Tab';

const APPEARANCE_STYLES = 'border-r border-subtle text-muted text-sm';
const BUTTON_SIZING_STYLES = 'w-[44px] flex-shrink-0';
const HOVER_STYLES = 'hover:bg-canvasSubtle';
const LAYOUT_STYLES = 'flex h-[40px] items-center justify-center relative';

interface BaseIconTabProps {
  className?: string;
  icon: React.ReactNode;
}

// Button variant (no value, never appears active)
interface IconTabButtonProps
  extends BaseIconTabProps,
    Omit<React.ComponentPropsWithoutRef<'button'>, 'value'> {
  value?: never;
}

// Tab variant (value required, can be active)
interface IconTabTabProps extends BaseIconTabProps, Omit<TabProps, 'title' | 'iconBefore'> {
  title?: never;
  value: string;
}

function isTabVariant(props: IconTabProps): props is IconTabTabProps {
  return props.value !== undefined;
}

export type IconTabProps = IconTabButtonProps | IconTabTabProps;

export const IconTab = forwardRef<HTMLButtonElement | React.ElementRef<typeof Tab>, IconTabProps>(
  (props, ref) => {
    if (isTabVariant(props)) {
      const { icon, ...tabProps } = props;
      return (
        <Tab
          className={cn(
            'w-[44px] min-w-[44px] max-w-[44px] justify-center gap-0 px-0',
            props.className
          )}
          disallowClose
          iconBefore={icon}
          ref={ref}
          {...tabProps}
        />
      );
    }

    const { icon, ...buttonProps } = props;
    return (
      <button
        className={cn(
          APPEARANCE_STYLES,
          BUTTON_SIZING_STYLES,
          HOVER_STYLES,
          LAYOUT_STYLES,
          props.className
        )}
        ref={ref}
        type="button"
        {...buttonProps}
      >
        <span className="flex-shrink-0">{icon}</span>
      </button>
    );
  }
);
