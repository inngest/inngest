import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

import { Pill } from './Pill';

type FunctionsCellContentProps = {
  pills: React.ReactNode[];
  alwaysVisibleCount?: number;
};

export function HorizontalPillList({ pills, alwaysVisibleCount }: FunctionsCellContentProps) {
  if (pills.length === 0) return null;

  if (alwaysVisibleCount && pills.length > alwaysVisibleCount) {
    const hiddenPills = pills.slice(alwaysVisibleCount);
    const alwaysVisiblePills = pills.slice(0, alwaysVisibleCount);

    return (
      <div className="flex items-center gap-1">
        {alwaysVisiblePills}

        <Tooltip delayDuration={0}>
          <TooltipTrigger className="flex cursor-default">
            <Pill className="px-2.5 align-middle">+{hiddenPills.length}</Pill>
          </TooltipTrigger>

          <TooltipContent sideOffset={5} className="p-3">
            <div className="flex flex-col gap-2">{hiddenPills}</div>
          </TooltipContent>
        </Tooltip>
      </div>
    );
  }

  return <div className="flex items-center gap-1">{pills}</div>;
}
