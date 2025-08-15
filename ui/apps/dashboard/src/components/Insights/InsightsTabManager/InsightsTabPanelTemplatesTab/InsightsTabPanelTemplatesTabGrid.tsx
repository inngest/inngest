'use client';

import { InsightsTabPanelTemplatesTabCard } from './InsightsTabPanelTemplatesTabCard';
import { TEMPLATES } from './templates';

export function InsightsTabPanelTemplatesTabGrid() {
  return (
    <div className="flex flex-wrap gap-6 pb-12">
      {TEMPLATES.map((template) => (
        <InsightsTabPanelTemplatesTabCard key={template.id} template={template} />
      ))}
    </div>
  );
}
