import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

import { Pill } from './Pill';

type Props = {
  eventCount: number;
};

export function BatchSize({ eventCount }: Props) {
  let tooltipContent = `Batch contains ${eventCount} events`;
  if (eventCount === 1) {
    tooltipContent = 'Batch contains 1 event';
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="flex h-fit">
          <Pill className="border-muted text-subtle border px-3 py-1" flatSide="right">
            <span>Batch</span>
          </Pill>
          <Pill className="border-muted text-subtle border px-3 py-1" flatSide="left">
            <span>{eventCount}</span>
          </Pill>
        </div>
      </TooltipTrigger>
      <TooltipContent className="font-mono text-xs">{tooltipContent}</TooltipContent>
    </Tooltip>
  );
}
