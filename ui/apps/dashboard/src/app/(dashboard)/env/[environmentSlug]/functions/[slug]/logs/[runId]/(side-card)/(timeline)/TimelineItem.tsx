import { capitalCase } from 'change-case';
import { type JsonValue } from 'type-fest';

import { maxRenderedOutputSizeBytes } from '@/app/consts';
import { Alert } from '@/components/Alert';
import SyntaxHighlighter from '@/components/SyntaxHighlighter';
import { Time } from '@/components/Time';
import { RunHistoryType } from '@/gql/graphql';
import CompletedIcon from '@/icons/timeline-item-completed.svg';
import DefaultIcon from '@/icons/timeline-item-default.svg';
import FailedIcon from '@/icons/timeline-item-failed.svg';
import Scheduled from '@/icons/timeline-item-scheduled.svg';
import Sleeping from '@/icons/timeline-item-sleeping.svg';
import WaitingIcon from '@/icons/timeline-item-waiting.svg';
import cn from '@/utils/cn';

const timelineItemTypeIcons = {
  [RunHistoryType.EventReceived]: DefaultIcon,
  [RunHistoryType.FunctionCancelled]: FailedIcon,
  [RunHistoryType.FunctionCompleted]: CompletedIcon,
  [RunHistoryType.FunctionFailed]: FailedIcon,
  [RunHistoryType.FunctionStarted]: DefaultIcon,
  [RunHistoryType.FunctionScheduled]: WaitingIcon,
  [RunHistoryType.StepCompleted]: CompletedIcon,
  [RunHistoryType.StepErrored]: FailedIcon,
  [RunHistoryType.StepFailed]: FailedIcon,
  [RunHistoryType.StepScheduled]: Scheduled,
  [RunHistoryType.StepSleeping]: Sleeping,
  [RunHistoryType.StepStarted]: DefaultIcon,
  [RunHistoryType.StepWaiting]: WaitingIcon,
  [RunHistoryType.Unknown]: DefaultIcon,
} as const satisfies Record<RunHistoryType, SVGComponent>;

type TimelineItemProps = {
  item: {
    id?: string;
    type: RunHistoryType;
    createdAt?: string;
    output?: string | null;
  };
  isFirst?: boolean;
  isLast?: boolean;
  // isNested represents whether this is a nested item for read
  isNested?: boolean;
  previousTime?: string;
};

export default function TimelineItem({ item, isFirst = false, isLast = false }: TimelineItemProps) {
  let parsedOutput: string | JsonValue = '';
  let isOutputTooLarge = false;
  if (typeof item.output === 'string') {
    // Keeps the tab from crashing when the output is huge.
    if (item.output.length > maxRenderedOutputSizeBytes) {
      isOutputTooLarge = true;
    } else {
      try {
        parsedOutput = JSON.parse(item.output);
      } catch (error) {
        console.error(`Error parsing JSON output of timeline item ${item.id}: `, error);
        parsedOutput = item.output;
      }
    }
  }

  const TimelineItemTypeIcon = timelineItemTypeIcons[item.type] ?? DefaultIcon;

  return (
    <li>
      <div className={cn('relative pb-2', !isFirst && 'pt-5')}>
        <span
          className={cn(
            'absolute left-4 ml-px w-0.5 bg-slate-700',
            isLast ? 'h-8' : 'h-full',
            isFirst ? 'top-4' : 'top-0'
          )}
          aria-hidden="true"
        />
        <div className="relative flex items-start">
          <div className="px-2">
            <div className="flex h-5 w-5 items-center justify-center">
              <TimelineItemTypeIcon aria-hidden="true" className="h-3.5 w-3.5" />
            </div>
          </div>
          <div className="flex min-w-0 flex-1 justify-between pr-4 text-[13px] font-light">
            <span className="text-[13px]">{capitalCase(item.type.replace('STEP_', ''))}</span>
            {item.createdAt && <Time value={new Date(item.createdAt)} />}
          </div>
        </div>

        {isOutputTooLarge && (
          <Alert className="relative z-10 mt-3 w-fit p-2" severity="warning">
            Output size is too large to render
          </Alert>
        )}

        {parsedOutput && (
          <div className="relative z-10 mt-3 p-2">
            <div className="bg-slate-940 rounded-lg p-6">
              <SyntaxHighlighter language="json" className="text-xs">
                {JSON.stringify(parsedOutput, null, 2)}
              </SyntaxHighlighter>
            </div>
          </div>
        )}
      </div>
    </li>
  );
}
