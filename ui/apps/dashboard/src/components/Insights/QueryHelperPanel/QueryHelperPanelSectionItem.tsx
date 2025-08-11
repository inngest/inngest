'use client';

import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';

import type { Query } from './types';

interface QueryHelperPanelSectionItemProps {
  onQuerySelect: (query: Query) => void;
  query: Query;
}

export function QueryHelperPanelSectionItem({
  query,
  onQuerySelect,
}: QueryHelperPanelSectionItemProps) {
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  const displayText = query.name;

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
          onQuerySelect(query);
        }}
        ref={buttonRef}
      >
        {displayText}
      </button>
    </OptionalTooltip>
  );
}
