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
    <AccordionList.Item className="border-0" value={title}>
      <AccordionList.Trigger className="data-[state=open]:border-0">
        <span className="text-light text-xs font-medium">{title}</span>
      </AccordionList.Trigger>
      <AccordionList.Content className="!py-1">
        <QueryHelperPanelSectionContent onQuerySelect={onQuerySelect} queries={queries} />
      </AccordionList.Content>
    </AccordionList.Item>
  );
}
