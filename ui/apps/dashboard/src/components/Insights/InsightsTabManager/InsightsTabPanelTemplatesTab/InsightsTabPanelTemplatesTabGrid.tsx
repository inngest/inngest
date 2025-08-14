'use client';

import type { TabManagerActions } from '../InsightsTabManager';
import { InsightsTabPanelTemplatesTabCard } from './InsightsTabPanelTemplatesTabCard';
import { TEMPLATES } from './templates';

interface InsightsTabPanelTemplatesTabGridProps {
  tabManagerActions: TabManagerActions;
}

export function InsightsTabPanelTemplatesTabGrid({
  tabManagerActions,
}: InsightsTabPanelTemplatesTabGridProps) {
  return (
    <div className="flex flex-wrap gap-6 pb-12">
      {TEMPLATES.map((template) => (
        <InsightsTabPanelTemplatesTabCard
          key={template.id}
          tabManagerActions={tabManagerActions}
          template={template}
        />
      ))}
    </div>
  );
}
