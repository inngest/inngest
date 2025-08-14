'use client';

import { InsightsTabPanelTemplatesTabGrid } from './InsightsTabPanelTemplatesTabGrid';

export function InsightsTabPanelTemplatesTabLeft() {
  return (
    <div className="flex flex-1 flex-col">
      <div className="mb-10">
        <h2 className="text-basis mb-1 text-xl">Start with a template</h2>
        <p className="text-muted text-sm">Choose a template to start exploring your data</p>
      </div>
      <InsightsTabPanelTemplatesTabGrid />
    </div>
  );
}
