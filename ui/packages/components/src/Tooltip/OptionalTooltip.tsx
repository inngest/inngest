import type { ReactNode } from 'react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

export const OptionalTooltip = ({
  children,
  tooltip,
  side = 'right',
}: {
  children: ReactNode;
  tooltip?: ReactNode;
  side?: 'top' | 'bottom' | 'right' | 'left';
}) =>
  tooltip ? (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent
        side={side}
        className="text-muted flex h-8 items-center px-4 text-xs leading-[18px]"
      >
        {tooltip}
      </TooltipContent>
    </Tooltip>
  ) : (
    children
  );
