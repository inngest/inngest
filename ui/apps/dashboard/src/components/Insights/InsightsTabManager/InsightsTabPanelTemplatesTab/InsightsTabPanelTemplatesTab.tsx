'use client';

import { InsightsTabPanelTemplatesTabLeft } from './InsightsTabPanelTemplatesTabLeft';
import { InsightsTabPanelTemplatesTabRight } from './InsightsTabPanelTemplatesTabRight';

export function InsightsTabPanelTemplatesTab() {
  return (
    <div className="col-span-1 row-span-2 flex h-full w-full gap-12 overflow-y-auto p-12">
      <InsightsTabPanelTemplatesTabLeft />
      <InsightsTabPanelTemplatesTabRight />
    </div>
  );
}
