'use client';

import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';

import { QueryHelperPanelSectionContent } from './QueryHelperPanelSectionContent';
import type { Query } from './types';

interface QueryHelperPanelSectionProps {
  onQuerySelect: (query: Query) => void;
  queries: {
    data: undefined | Query[];
    error: undefined | string;
    isLoading: boolean;
  };
  title: string;
}

export function QueryHelperPanelSection({
  onQuerySelect,
  queries,
  title,
}: QueryHelperPanelSectionProps) {
  return (
    <AccordionList.Item value={title}>
      <AccordionList.Trigger>
        <span className="text-light text-xs font-medium">{title}</span>
      </AccordionList.Trigger>
      <AccordionList.Content>
        <QueryHelperPanelSectionContent onQuerySelect={onQuerySelect} queries={queries} />
      </AccordionList.Content>
    </AccordionList.Item>
  );
}
