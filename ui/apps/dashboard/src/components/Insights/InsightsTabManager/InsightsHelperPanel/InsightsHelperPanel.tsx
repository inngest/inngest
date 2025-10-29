'use client';

import { useMemo } from 'react';
import { RiCloseLine } from '@remixicon/react';

import { InsightsChat } from '../../InsightsChat/InsightsChat';
import { InsightsHelperPanelIcon } from './InsightsHelperPanelIcon';
import {
  DOCUMENTATION,
  INSIGHTS_AI,
  SCHEMA_EXPLORER,
  SUPPORT,
  type HelperTitle,
} from './constants';

type InsightsHelperPanelProps = {
  active: HelperTitle;
  agentThreadId?: string;
  onClose: () => void;
};

export function InsightsHelperPanel({ active, agentThreadId, onClose }: InsightsHelperPanelProps) {
  const content = useMemo(() => {
    switch (active) {
      case INSIGHTS_AI: {
        if (!agentThreadId) return null;
        return <InsightsChat agentThreadId={agentThreadId} />;
      }
      case DOCUMENTATION:
        return <div className="text-sm">Docs helper (placeholder)</div>;
      case SCHEMA_EXPLORER:
        return <div className="text-sm">Schemas helper (placeholder)</div>;
      case SUPPORT:
        return <div className="text-sm">Support helper (placeholder)</div>;
      default:
        return null;
    }
  }, [active, agentThreadId]);

  if (content === null) return null;

  return (
    <div className="flex h-full w-full flex-col">
      <div className="border-subtle flex h-[49px] shrink-0 flex-row items-center justify-between border-b px-3">
        <div className="flex flex-row items-center gap-2">
          <InsightsHelperPanelIcon className="text-subtle" title={active} />
          <div className="text-muted text-sm font-normal uppercase tracking-wider">{active}</div>
        </div>
        <button
          aria-label="Close helper"
          className="hover:bg-canvasSubtle hover:text-basis text-subtle flex h-8 w-8 items-center justify-center rounded-md transition-colors"
          onClick={onClose}
          type="button"
        >
          <RiCloseLine size={18} />
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">{content}</div>
    </div>
  );
}
