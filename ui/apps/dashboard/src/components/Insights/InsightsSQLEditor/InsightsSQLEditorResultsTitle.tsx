'use client';

import { Pill } from '@inngest/components/Pill';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

export function InsightsSQLEditorResultsTitle() {
  return (
    <div className="mr-1 flex items-center gap-2">
      <span>Results</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Pill appearance="outlined" kind="info">
              LIMIT 1000
            </Pill>
          </span>
        </TooltipTrigger>
        <TooltipContent className="p-2 text-xs" side="right" sideOffset={3}>
          Results are currently limited to at most 1000 rows.
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
