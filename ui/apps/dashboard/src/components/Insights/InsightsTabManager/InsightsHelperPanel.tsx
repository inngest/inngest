'use client';

import { useMemo } from 'react';

import { InsightsChat } from '../InsightsChat/InsightsChat';

type InsightsHelperPanelProps = {
  active: null | string;
  agentThreadId?: string;
};

export function InsightsHelperPanel({ active, agentThreadId }: InsightsHelperPanelProps) {
  const content = useMemo(() => {
    switch (active) {
      case 'AI':
        if (!agentThreadId) return null;
        return <InsightsChat agentThreadId={agentThreadId} onToggleChat={() => {}} />;
      case 'Docs':
        return <div className="text-sm">Docs helper (placeholder)</div>;
      case 'Schemas':
        return <div className="text-sm">Schemas helper (placeholder)</div>;
      case 'Support':
        return <div className="text-sm">Support helper (placeholder)</div>;
      default:
        return null;
    }
  }, [active, agentThreadId]);

  if (content === null) return null;

  return <div className="h-full w-full overflow-auto p-3">{content}</div>;
}
