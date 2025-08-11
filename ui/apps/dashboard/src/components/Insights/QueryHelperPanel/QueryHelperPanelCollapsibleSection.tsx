'use client';

import { useState } from 'react';
import { RiArrowDownSLine } from '@remixicon/react';

import { QueryHelperPanelSectionContent } from './QueryHelperPanelSectionContent';
import type { Query } from './types';

interface QueryHelperPanelCollapsibleSectionProps {
  onQuerySelect: (query: Query) => void;
  queries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  title: string;
}

export function QueryHelperPanelCollapsibleSection({
  onQuerySelect,
  queries,
  title,
}: QueryHelperPanelCollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(true);

  return (
    <div>
      <button
        className="hover:bg-canvasSubtle flex w-full items-center justify-between px-3 py-2 text-left transition-colors"
        onClick={() => setIsOpen(!isOpen)}
      >
        <span className="text-light text-xs font-medium">{title}</span>
        <RiArrowDownSLine
          className={`h-4 w-4 transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
        />
      </button>
      {isOpen && (
        <div className="px-4 pb-3">
          <QueryHelperPanelSectionContent onQuerySelect={onQuerySelect} queries={queries} />
        </div>
      )}
    </div>
  );
}
