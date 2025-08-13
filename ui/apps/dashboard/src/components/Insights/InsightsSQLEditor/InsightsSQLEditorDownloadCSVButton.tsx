'use client';

import { Button } from '@inngest/components/Button/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

export function InsightsSQLEditorDownloadCSVButton() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            appearance="ghost"
            disabled
            kind="secondary"
            label="Download as .csv"
            size="medium"
          />
        </span>
      </TooltipTrigger>
      <TooltipContent>Coming soon</TooltipContent>
    </Tooltip>
  );
}
