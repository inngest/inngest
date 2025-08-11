'use client';

import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiBookmarkLine, RiHistoryLine } from '@remixicon/react';

import type { Query } from './types';

interface QueryHelperPanelSectionItemProps {
  onQuerySelect: (query: Query) => void;
  query: Query;
  sectionType: 'history' | 'saved';
}

export function QueryHelperPanelSectionItem({
  query,
  onQuerySelect,
  sectionType,
}: QueryHelperPanelSectionItemProps) {
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  const displayText = query.name;
  const Icon = sectionType === 'saved' ? RiBookmarkLine : RiHistoryLine;

  useEffect(() => {
    const el = buttonRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [displayText]);

  return (
    <OptionalTooltip side="right" tooltip={isTruncated ? displayText : ''}>
      <button
        className="hover:bg-canvasSubtle text-subtle flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors"
        onClick={() => {
          onQuerySelect(query);
        }}
        ref={buttonRef}
      >
        <Icon className="h-4 w-4 flex-shrink-0" />
        <span className="overflow-hidden truncate text-ellipsis whitespace-nowrap">
          {displayText}
        </span>
      </button>
    </OptionalTooltip>
  );
}
