import { Link } from '@inngest/components/Link/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiQuestionLine } from '@remixicon/react';

export const EventInfo = () => (
  <Tooltip>
    <TooltipTrigger>
      <RiQuestionLine className="text-subtle h-[18px] w-[18px]" />
    </TooltipTrigger>
    <TooltipContent
      side="right"
      sideOffset={2}
      className="border-muted text-muted mt-6 flex flex-col rounded-md border p-0 text-sm"
    >
      <div className="border-subtle border-b px-4 py-2 ">
        List of all Inngest events in the current environment.
      </div>

      <div className="px-4 py-2">
        <Link href={'https://www.inngest.com/docs/events'} target="_blank">
          Learn how events work
        </Link>
      </div>
    </TooltipContent>
  </Tooltip>
);
