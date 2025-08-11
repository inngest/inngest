'use client';

import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiAddLine, RiHome5Line } from '@remixicon/react';

interface InsightsTabActionIconProps {
  isActive?: boolean;
  isFirst?: boolean;
  onClick: () => void;
  tooltip?: string;
  type: 'home' | 'add';
}

export function InsightsTabActionIcon({
  isActive = false,
  isFirst = false,
  onClick,
  tooltip,
  type,
}: InsightsTabActionIconProps) {
  const Icon = type === 'home' ? RiHome5Line : RiAddLine;

  return (
    <OptionalTooltip side="top" tooltip={tooltip || ''}>
      <button
        className={cn(
          'bg-canvasBase border-subtle hover:bg-canvasMuted hover:border-muted relative flex h-12 w-12 items-center justify-center border-r transition-colors',
          !isFirst && 'border-l'
        )}
        onClick={onClick}
      >
        <Icon className="h-4 w-4 text-slate-500" />
        {isActive && <div className="bg-primary-intense absolute bottom-0 left-0 right-0 h-0.5" />}
      </button>
    </OptionalTooltip>
  );
}
