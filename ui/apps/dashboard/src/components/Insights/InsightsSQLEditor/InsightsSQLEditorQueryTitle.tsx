'use client';

import { useEffect, useRef, useState } from 'react';
import { Input } from '@inngest/components/Forms/Input';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { Query } from '@/components/Insights/types';

type InsightsSQLEditorQueryTitleProps = {
  tab: Query;
};

export function InsightsSQLEditorQueryTitle({ tab }: InsightsSQLEditorQueryTitleProps) {
  const { onNameChange, queryName } = useInsightsStateMachineContext();
  const [isEditing, setIsEditing] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const [isTruncated, setIsTruncated] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const textRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    // Auto-focus and select all text when entering edit mode
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }

    if (!isEditing) {
      const el = textRef.current;
      if (el === null) return;

      setIsTruncated(el.scrollWidth > el.clientWidth);
    }
  }, [isEditing]);

  if (isEditing) {
    return (
      <Input
        className="mr-2 w-[314px]"
        name={`${tab.id}-query-title`}
        onBlur={() => {
          setIsEditing(false);
          setIsHovered(false);
        }}
        onChange={(e) => onNameChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === 'Escape') {
            e.preventDefault();
            inputRef.current?.blur();
          }
        }}
        ref={inputRef}
        value={queryName}
      />
    );
  }

  return (
    <OptionalTooltip side="bottom" tooltip={isTruncated ? queryName : ''}>
      <div
        className={cn(
          'text-basis mr-2 flex h-8 w-[314px] cursor-pointer items-center rounded px-2 py-2 text-sm normal-case leading-normal transition-all duration-150',
          isHovered
            ? 'bg-canvasSubtle border-muted border'
            : 'border border-transparent bg-transparent'
        )}
        onClick={() => {
          setIsEditing(true);
        }}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <span ref={textRef} className="truncate whitespace-nowrap">
          {queryName}
        </span>
      </div>
    </OptionalTooltip>
  );
}
