'use client';

import { Pill } from '@inngest/components/Pill';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

const ROW_LIMIT = 1000;

export function InsightsSQLEditorResultsTitle() {
  return (
    <div className="mr-1 flex items-center gap-2">
      <span className="uppercase">Results</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Pill appearance="outlined" kind="info">
              Row limit : {ROW_LIMIT}
            </Pill>
          </span>
        </TooltipTrigger>
        <TooltipContent className="p-2 text-xs" side="right" sideOffset={3}>
          Results are currently limited to at most {ROW_LIMIT} rows.
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
