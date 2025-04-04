import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

import { Pill } from './Pill';

type FunctionsCellContentProps = {
  pills: React.ReactNode[];
  alwaysVisibleCount?: number;
};

export function HorizontalPillList({ pills, alwaysVisibleCount }: FunctionsCellContentProps) {
  if (pills.length === 0) return null;

  // If no alwaysVisibleCount is specified or there aren't more pills than the limit, show all
  if (!alwaysVisibleCount || pills.length <= alwaysVisibleCount) {
    return (
      <div className="flex items-center gap-1">
        {pills.map((pill, index) => (
          <div key={index} className="min-w-0 overflow-hidden">
            {pill}
          </div>
        ))}
      </div>
    );
  }

  // If we have more pills than alwaysVisibleCount, use the "+X" condensed view
  const hiddenPills = pills.slice(alwaysVisibleCount);
  const alwaysVisiblePills = pills.slice(0, alwaysVisibleCount);

  return (
    <div className="flex items-center gap-1">
      {alwaysVisiblePills.map((pill, index) => (
        <div key={index} className="min-w-0 overflow-hidden">
          {pill}
        </div>
      ))}

      <Tooltip delayDuration={0}>
        <TooltipTrigger className="flex flex-shrink-0 cursor-default">
          <Pill className="px-2.5 align-middle">+{hiddenPills.length}</Pill>
        </TooltipTrigger>

        <TooltipContent sideOffset={5} className="p-3" side="bottom">
          <div className="flex flex-col gap-2">{hiddenPills}</div>
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
