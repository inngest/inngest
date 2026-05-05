import { type ReactElement, type ReactNode } from 'react';
import { RiCloseLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

type HelperPanelFrameProps = {
  title: string;
  icon?: ReactElement | null;
  onClose: () => void;
  children: ReactNode;
  /** Overflow handling for the content region. Defaults to scroll. */
  contentClassName?: string;
};

/**
 * Shared chrome (header with icon, title, close button) for panels controlled
 * by `HelperPanelControl`. Used by both the Insights and Experiments helper
 * panels so they stay visually identical.
 */
export function HelperPanelFrame({
  title,
  icon,
  onClose,
  children,
  contentClassName,
}: HelperPanelFrameProps) {
  return (
    <div className="flex h-full w-full flex-col">
      <div className="border-subtle flex h-[49px] shrink-0 flex-row items-center justify-between border-b px-3">
        <div className="flex flex-row items-center gap-2">
          {icon}
          <div className="text-sm font-normal">{title}</div>
        </div>
        <button
          aria-label="Close panel"
          className="hover:bg-canvasSubtle hover:text-basis text-subtle -mr-1 flex h-8 w-8 items-center justify-center rounded-md transition-colors"
          onClick={onClose}
          type="button"
        >
          <RiCloseLine size={18} />
        </button>
      </div>
      <div className={cn('min-h-0 flex-1 overflow-y-auto', contentClassName)}>{children}</div>
    </div>
  );
}
