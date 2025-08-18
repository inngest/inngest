'use client';

import { useEffect, useRef, useState } from 'react';
import { Input } from '@inngest/components/Forms/Input';
import { cn } from '@inngest/components/utils/classNames';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { TabConfig } from '../InsightsTabManager/InsightsTabManager';

type InsightsSQLEditorQueryTitleProps = {
  tab: TabConfig;
};

export function InsightsSQLEditorQueryTitle({ tab }: InsightsSQLEditorQueryTitleProps) {
  const { queryName, onNameChange } = useInsightsStateMachineContext();
  const [isEditing, setIsEditing] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-focus and select all text when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
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
    <div
      className={cn(
        'text-basis mr-2 flex h-8 w-[314px] cursor-pointer items-center rounded px-2 py-2 text-sm normal-case leading-none transition-all duration-150',
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
      {queryName}
    </div>
  );
}
