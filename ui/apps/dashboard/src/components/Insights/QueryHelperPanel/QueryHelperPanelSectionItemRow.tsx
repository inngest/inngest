'use client';

import { forwardRef, useEffect, useRef, useState, type ReactNode } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';

type QueryHelperPanelSectionItemRowProps = {
  icon: ReactNode;
  isActive: boolean;
  onContextMenu: (e: React.MouseEvent) => void;
  onClick?: (e: React.MouseEvent) => void;
  text: string;
};

export const QueryHelperPanelSectionItemRow = forwardRef<
  HTMLDivElement,
  QueryHelperPanelSectionItemRowProps
>(({ icon, isActive, onClick, onContextMenu, text }: QueryHelperPanelSectionItemRowProps, ref) => {
  const textRef = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  useEffect(() => {
    const el = textRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [text]);

  return (
    <div
      ref={ref}
      className={cn(
        'text-subtle flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors',
        isActive ? 'bg-canvasSubtle' : 'hover:bg-canvasSubtle'
      )}
      onClick={onClick}
      onContextMenu={onContextMenu}
    >
      {icon}
      <OptionalTooltip side="right" tooltip={isTruncated ? text : ''}>
        <span
          ref={textRef}
          className="flex-1 overflow-hidden truncate text-ellipsis whitespace-nowrap"
        >
          {text}
        </span>
      </OptionalTooltip>
    </div>
  );
});

QueryHelperPanelSectionItemRow.displayName = 'QueryHelperPanelSectionItemRow';
