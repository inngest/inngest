'use client';

import { forwardRef, useContext } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';
import { RiCloseLine } from '@remixicon/react';

import { TabsContext } from './TabsContext';

const ACTIVE_BORDER_STYLES =
  'data-[state=active]:after:absolute data-[state=active]:after:bg-contrast data-[state=active]:after:bottom-0 data-[state=active]:after:h-0.5 data-[state=active]:after:left-0 data-[state=active]:after:right-0';
const ACTIVE_TEXT_STYLES = 'data-[state=active]:text-basis';
const APPEARANCE_STYLES = 'border-r border-subtle text-muted text-sm';
const HOVER_STYLES = 'hover:bg-canvasSubtle';
const LAYOUT_STYLES = 'flex flex-1 h-[40px] items-center relative';
const SIZING_STYLES = 'max-w-[200px] min-w-[84px]';
const SPACING_STYLES = 'gap-1.5 px-3';

// TODO: Add overflow tooltip functionality similar to approach from Pill component.

export interface TabProps
  extends Omit<React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>, 'children'> {
  disallowClose?: boolean;
  iconBefore?: React.ReactNode;
  title: string;
}

export const Tab = forwardRef<React.ElementRef<typeof TabsPrimitive.Trigger>, TabProps>(
  ({ className, disallowClose, iconBefore, title, value, ...props }, ref) => {
    const { defaultIconBefore, onClose } = useContext(TabsContext);

    const finalIconBefore = iconBefore ?? defaultIconBefore;

    return (
      <TabsPrimitive.Trigger
        className={cn(
          ACTIVE_BORDER_STYLES,
          ACTIVE_TEXT_STYLES,
          APPEARANCE_STYLES,
          HOVER_STYLES,
          LAYOUT_STYLES,
          SIZING_STYLES,
          SPACING_STYLES,
          className
        )}
        ref={ref}
        value={value}
        {...props}
      >
        {finalIconBefore && <span className="flex-shrink-0">{finalIconBefore}</span>}
        {title && <span className="flex-1 truncate text-left">{title}</span>}
        {onClose && !disallowClose && (
          <span
            className="p-0.5"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onClose(value);
            }}
          >
            <RiCloseLine size={14} />
          </span>
        )}
      </TabsPrimitive.Trigger>
    );
  }
);
