import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiQuestionLine } from '@remixicon/react';

export function BillingBannerTooltip({ children }: React.PropsWithChildren<{}>) {
  return (
    <Tooltip>
      <TooltipTrigger>
        <RiQuestionLine className="text-subtle mx-1 h-[18px] w-[18px]" />
      </TooltipTrigger>

      <TooltipContent
        // High z-index is necessary to prevent cutoff from the page's normal
        // content header.
        className="z-[110]"
      >
        {children}
      </TooltipContent>
    </Tooltip>
  );
}
