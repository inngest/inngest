'use client';

import type { TabManagerActions } from '../InsightsTabManager';
import { InsightsTabPanelTemplatesTabLeft } from './InsightsTabPanelTemplatesTabLeft';
import { InsightsTabPanelTemplatesTabRight } from './InsightsTabPanelTemplatesTabRight';

interface InsightsTabPanelTemplatesTabProps {
  tabManagerActions: TabManagerActions;
}

export function InsightsTabPanelTemplatesTab({
  tabManagerActions,
}: InsightsTabPanelTemplatesTabProps) {
  return (
    <div className="col-span-1 row-span-2 flex h-full w-full gap-12 overflow-y-auto p-12">
      <InsightsTabPanelTemplatesTabLeft tabManagerActions={tabManagerActions} />
      <InsightsTabPanelTemplatesTabRight tabManagerActions={tabManagerActions} />
    </div>
  );
}
