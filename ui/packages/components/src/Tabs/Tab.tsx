import { forwardRef, useContext, useEffect, useRef, useState } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import * as TabsPrimitive from '@radix-ui/react-tabs';
import { RiCloseLine } from '@remixicon/react';

import { TabsContext } from './TabsContext';

const ACTIVE_BORDER_STYLES =
  'data-[state=active]:after:absolute data-[state=active]:after:bg-contrast data-[state=active]:after:bottom-0 data-[state=active]:after:h-0.5 data-[state=active]:after:left-0 data-[state=active]:after:right-0';
const ACTIVE_TEXT_STYLES = 'data-[state=active]:text-basis';
const APPEARANCE_STYLES = 'bg-canvasBase border-r border-subtle text-muted text-sm';
const CLOSE_BUTTON_STYLES = 'cursor-pointer flex-shrink-0 hover:bg-muted p-0.5 rounded';
const FOCUS_STYLES =
  'focus:outline-none focus-visible:outline-none focus-visible:ring-0 outline-none';
const HOVER_STYLES = 'hover:bg-canvasSubtle';
const INTERACTION_STYLES = 'group transition-all';
const LAYOUT_STYLES = 'flex flex-1 h-[40px] items-center relative';
const SIZING_STYLES = 'max-w-[200px] min-w-[84px]';
const SPACING_STYLES = 'gap-1.5 px-3 py-2';

export interface TabProps extends React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger> {
  iconBefore?: React.ReactNode;
}

export const Tab = forwardRef<React.ElementRef<typeof TabsPrimitive.Trigger>, TabProps>(
  ({ children, className, iconBefore, value, ...props }, ref) => {
    const { defaultIconBefore, onClose } = useContext(TabsContext);
    const { isOverflowing, textRef } = useOverflowTooltip(children);

    const finalIconBefore = iconBefore ?? defaultIconBefore;

    return (
      <TabsPrimitive.Trigger
        className={cn(
          ACTIVE_BORDER_STYLES,
          ACTIVE_TEXT_STYLES,
          APPEARANCE_STYLES,
          FOCUS_STYLES,
          HOVER_STYLES,
          INTERACTION_STYLES,
          LAYOUT_STYLES,
          SIZING_STYLES,
          SPACING_STYLES,
          className
        )}
        ref={ref}
        title={isOverflowing && typeof children === 'string' ? children : undefined}
        value={value}
        {...props}
      >
        {finalIconBefore && <span className="flex-shrink-0">{finalIconBefore}</span>}
        <span ref={textRef} className="flex-1 truncate text-left">
          {children}
        </span>
        {onClose && (
          <span
            className={CLOSE_BUTTON_STYLES}
            onMouseDown={(e) => {
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

function useOverflowTooltip(content: React.ReactNode) {
  const [isOverflowing, setIsOverflowing] = useState(false);
  const textRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    const element = textRef.current;
    if (!element) return;

    const checkOverflow = () => {
      setIsOverflowing(element.scrollWidth > element.clientWidth);
    };

    checkOverflow();
    const resizeObserver = new ResizeObserver(checkOverflow);
    resizeObserver.observe(element);

    return () => {
      resizeObserver.disconnect();
    };
  }, [content]);

  return { isOverflowing, textRef };
}
