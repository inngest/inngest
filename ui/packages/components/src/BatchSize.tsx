import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

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
        <div className="flex text-slate-400">
          <div className="rounded-l-xl bg-slate-800 px-3 py-1">Batch</div>
          <div className="rounded-r-xl bg-slate-700 px-3 py-1">{eventCount}</div>
        </div>
      </TooltipTrigger>
      <TooltipContent className="font-mono text-xs">{tooltipContent}</TooltipContent>
    </Tooltip>
  );
}
