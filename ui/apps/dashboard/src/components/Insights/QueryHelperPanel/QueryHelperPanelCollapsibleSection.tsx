'use client';

import { useState } from 'react';
import { RiArrowDownSLine } from '@remixicon/react';

import {
  QueryHelperPanelSectionContent,
  type QueryHelperPanelSectionContentProps,
} from './QueryHelperPanelSectionContent';

interface QueryHelperPanelCollapsibleSectionProps extends QueryHelperPanelSectionContentProps {
  title: string;
}

export function QueryHelperPanelCollapsibleSection({
  onQuerySelect,
  queries,
  title,
  sectionType,
}: QueryHelperPanelCollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(true);

  return (
    <div className="px-4 pb-3">
      <button
        className="group flex w-full items-center justify-between py-2 text-left transition-colors"
        onClick={() => setIsOpen(!isOpen)}
      >
        <span className="text-muted group-hover:text-basis text-xs font-medium transition-colors">
          {title}
        </span>
        <RiArrowDownSLine
          className={`text-muted group-hover:text-basis h-4 w-4 transition-all duration-200 ${
            isOpen ? 'rotate-180' : ''
          }`}
        />
      </button>
      {isOpen && (
        <QueryHelperPanelSectionContent
          onQuerySelect={onQuerySelect}
          queries={queries}
          sectionType={sectionType}
        />
      )}
    </div>
  );
}
