import { useMemo } from 'react';
import { HelperPanelFrame } from '@inngest/components/HelperPanelControl';

import { InsightsHelperPanelIcon } from './InsightsHelperPanelIcon';
import {
  CELL_DETAIL,
  DOCUMENTATION,
  INSIGHTS_AI,
  SCHEMA_EXPLORER,
  SUPPORT,
  type HelperTitle,
} from './constants';
import { CellDetailView } from './features/CellDetail/CellDetailView';
import { InsightsChat } from './features/InsightsChat/InsightsChat';
import { SchemaExplorer } from './features/SchemaExplorer/SchemaExplorer';

type InsightsHelperPanelProps = {
  active: HelperTitle;
  agentThreadId?: string;
  onClose: () => void;
};

export function InsightsHelperPanel({
  active,
  agentThreadId,
  onClose,
}: InsightsHelperPanelProps) {
  const content = useMemo(() => {
    switch (active) {
      case INSIGHTS_AI: {
        if (!agentThreadId) return null;
        return <InsightsChat agentThreadId={agentThreadId} />;
      }
      case CELL_DETAIL:
        return <CellDetailView />;
      case DOCUMENTATION:
        return <div className="text-sm">Docs helper (placeholder)</div>;
      case SCHEMA_EXPLORER:
        return <SchemaExplorer />;
      case SUPPORT:
        return <div className="text-sm">Support helper (placeholder)</div>;
      default:
        return null;
    }
  }, [active, agentThreadId]);

  if (content === null) return null;

  return (
    <HelperPanelFrame
      title={active}
      icon={<InsightsHelperPanelIcon className="text-subtle" title={active} />}
      onClose={onClose}
      contentClassName="overflow-hidden"
    >
      {content}
    </HelperPanelFrame>
  );
}
