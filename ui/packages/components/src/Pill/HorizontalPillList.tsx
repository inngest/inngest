import { useState } from 'react';
import { Popover, PopoverContent, PopoverTrigger } from '@inngest/components/Popover';

import { Pill } from './Pill';

type FunctionsCellContentProps = {
  pills: React.ReactNode[];
  alwaysVisibleCount?: number;
};

export function HorizontalPillList({ pills, alwaysVisibleCount }: FunctionsCellContentProps) {
  if (pills.length === 0) return null;
  const [open, setOpen] = useState(false);

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

      <Popover open={open} onOpenChange={setOpen} modal={true}>
        <PopoverTrigger className="flex flex-shrink-0 cursor-pointer" asChild>
          <button
            onClick={(e) => {
              e.stopPropagation();
              setOpen(true);
            }}
          >
            <Pill appearance="outlined" className="hover:bg-canvasMuted rounded align-middle">
              + {hiddenPills.length} more
            </Pill>
          </button>
        </PopoverTrigger>

        <PopoverContent sideOffset={5} className="border-subtle p-3" side="right">
          <div className="flex flex-col gap-2">{hiddenPills}</div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
