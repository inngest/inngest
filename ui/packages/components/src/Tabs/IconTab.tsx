import { forwardRef } from 'react';
import { cn } from '@inngest/components/utils/classNames';

const APPEARANCE_STYLES = 'bg-canvasBase border-r border-subtle text-muted';
const FOCUS_STYLES =
  'focus:outline-none focus-visible:outline-none focus-visible:ring-0 outline-none';
const HOVER_STYLES = 'hover:bg-canvasSubtle';
const INTERACTION_STYLES = 'group transition-all cursor-pointer';
const LAYOUT_STYLES = 'flex h-[40px] items-center justify-center relative';
const SIZING_STYLES = 'w-[44px] flex-shrink-0';

export interface IconTabProps extends React.ComponentPropsWithoutRef<'button'> {
  icon: React.ReactNode;
}

export const IconTab = forwardRef<HTMLButtonElement, IconTabProps>(
  ({ className, icon, ...props }, ref) => {
    return (
      <button
        className={cn(
          APPEARANCE_STYLES,
          FOCUS_STYLES,
          HOVER_STYLES,
          INTERACTION_STYLES,
          LAYOUT_STYLES,
          SIZING_STYLES,
          className
        )}
        ref={ref}
        type="button"
        {...props}
      >
        <span className="flex-shrink-0">{icon}</span>
      </button>
    );
  }
);
