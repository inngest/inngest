'use client';

import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';

import { useInsightsStateMachineContext } from '../InsightsStateMachineContext/InsightsStateMachineContext';
import type { RecentQuery, SavedQuery } from './types';

interface QueryHelperPanelSectionProps {
  queries: (RecentQuery | SavedQuery)[];
  title: string;
}

export function QueryHelperPanelSection({ queries, title }: QueryHelperPanelSectionProps) {
  return (
    <div className="flex flex-col gap-2 p-4">
      <h3 className="text-light text-xs font-medium">{title}</h3>
      <div className="flex flex-col gap-1">
        {queries.map((query) => (
          <QueryHelperPanelSectionItem key={query.id} query={query} />
        ))}
      </div>
    </div>
  );
}

function QueryHelperPanelSectionItem({ query }: { query: RecentQuery | SavedQuery }) {
  const { onChange } = useInsightsStateMachineContext();
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  const displayText = 'name' in query ? query.name : query.text;

  useEffect(() => {
    const el = buttonRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [displayText]);

  return (
    <OptionalTooltip side="right" tooltip={isTruncated ? displayText : ''}>
      <button
        className="hover:bg-canvasSubtle text-subtle w-full cursor-pointer overflow-x-hidden truncate text-ellipsis whitespace-nowrap rounded px-2 py-1.5 text-left text-sm transition-colors"
        onClick={() => {
          onChange(query.text);
        }}
        ref={buttonRef}
      >
        {displayText}
      </button>
    </OptionalTooltip>
  );
}
