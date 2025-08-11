'use client';

import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiCloseLine, RiCodeLine } from '@remixicon/react';

interface InsightsTabProps {
  isActive: boolean;
  isFirst?: boolean;
  name: string;
  onClose?: () => void;
  onClick: () => void;
  showCloseButton?: boolean;
}

export function InsightsTab({
  isActive,
  isFirst = false,
  name,
  onClose,
  onClick,
  showCloseButton = true,
}: InsightsTabProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  useEffect(() => {
    const el = textRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [name]);

  const handleCloseClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onClose?.();
  };

  return (
    <OptionalTooltip side="top" tooltip={isTruncated ? name : ''}>
      <button
        className={cn(
          'bg-canvasBase border-subtle hover:bg-canvasMuted hover:border-muted relative box-border flex h-12 w-[200px] items-center gap-2 border-b-2 border-r border-b-transparent px-2 transition-colors',
          !isFirst && 'border-l',
          isActive ? 'border-b-primary-intense hover:border-b-primary-intense' : 'border-b-subtle'
        )}
        onClick={onClick}
      >
        <RiCodeLine className="h-4 w-4 flex-shrink-0 text-slate-500" />
        <span ref={textRef} className="flex-1 overflow-hidden truncate text-left text-sm">
          {name}
        </span>
        {showCloseButton && (
          <span
            className="flex h-8 w-8 flex-shrink-0 cursor-pointer items-center justify-center rounded hover:bg-slate-200 dark:hover:bg-slate-700"
            onClick={handleCloseClick}
          >
            <RiCloseLine className="h-4 w-4 text-slate-500" />
          </span>
        )}
      </button>
    </OptionalTooltip>
  );
}
