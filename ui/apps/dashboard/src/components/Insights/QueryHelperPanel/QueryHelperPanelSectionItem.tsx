'use client';

import { useEffect, useRef, useState } from 'react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiBookmarkLine, RiHistoryLine } from '@remixicon/react';

import type { TabConfig } from '@/components/Insights/InsightsTabManager/InsightsTabManager';
import type { Query, QuerySnapshot } from '../types';

interface QueryHelperPanelSectionItemProps {
  activeTabId: string;
  onQuerySelect: (query: Query | QuerySnapshot) => void;
  query: Query | QuerySnapshot;
  sectionType: 'history' | 'saved';
  tabs: TabConfig[];
}

export function QueryHelperPanelSectionItem({
  activeTabId,
  query,
  onQuerySelect,
  sectionType,
  tabs,
}: QueryHelperPanelSectionItemProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  const displayText = query.name;
  const Icon = sectionType === 'saved' ? RiBookmarkLine : RiHistoryLine;

  const isActiveTab = getIsActiveTab(activeTabId, tabs, query);

  useEffect(() => {
    const el = textRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [displayText]);

  return (
    <OptionalTooltip side="right" tooltip={isTruncated ? displayText : ''}>
      <button
        className={`text-subtle flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors ${
          isActiveTab ? 'bg-canvasSubtle' : 'hover:bg-canvasSubtle'
        }`}
        onClick={() => {
          onQuerySelect(query);
        }}
      >
        <Icon className="h-4 w-4 flex-shrink-0" />
        <span ref={textRef} className="overflow-hidden truncate text-ellipsis whitespace-nowrap">
          {displayText}
        </span>
      </button>
    </OptionalTooltip>
  );
}

function getIsActiveTab(
  activeTabId: string,
  tabs: TabConfig[],
  query: Query | QuerySnapshot
): boolean {
  return tabs.find((tab) => tab.id === activeTabId && tab.savedQueryId === query.id) !== undefined;
}
