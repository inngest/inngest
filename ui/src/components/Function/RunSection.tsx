import { useEffect, useMemo, useState } from 'react';
import ms from 'ms';

import { usePrettyJson } from '../../hooks/usePrettyJson';
import {
  EventStatus,
  FunctionEventType,
  FunctionRunStatus,
  StepEventType,
  useGetFunctionRunQuery,
} from '../../store/generated';
import { selectRun } from '../../store/global';
import { useAppDispatch, useAppSelector } from '../../store/hooks';
import { BlankSlate } from '../Blank';
import Button from '../Button/Button';
import CodeBlock from '../Code/CodeBlock';
import ContentCard from '../Content/ContentCard';
import TimelineFuncProgress from '../Timeline/TimelineFuncProgress';
import TimelineRow from '../Timeline/TimelineRow';

interface FunctionRunSectionProps {
  runId: string | null | undefined;
}

export const FunctionRunSection = ({ runId }: FunctionRunSectionProps) => {
  const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetFunctionRunQuery(
    { id: runId || '' },
    { pollingInterval, skip: !runId, refetchOnMountOrArgChange: true },
  );
  const run = useMemo(() => query.data?.functionRun, [query.data?.functionRun]);
  const timeline = useMemo(() => normalizeSteps(run?.timeline || null), [run]);
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (!run?.event?.id) {
      return;
    }

    if (run.event.id !== selectedEvent) {
      dispatch(selectRun(null));
    }
  }, [selectedEvent, run?.event?.id]);

  if (query.isLoading) {
    return (
      <ContentCard date={0} id="">
        <div className="w-full h-full flex items-center justify-center p-8">
          <div className="opacity-75 italic">Loading...</div>
        </div>
      </ContentCard>
    );
  }

  if (!run) {
    return (
      <ContentCard date={0} id="">
        <BlankSlate
          imageUrl="/images/no-fn-selected.png"
          title="No function run selected"
          subtitle="Select a function run on the left to see a timeline of its execution."
        />
      </ContentCard>
    );
  }

  return (
    <ContentCard
      title={run.name || 'Unknown'}
      date={run.startedAt}
      id={run.id}
      idPrefix={'Run ID'}
      // button={<Button label="Open Function" icon={<IconFeed />} />}
    >
      <div className="flex justify-end px-4 border-t border-slate-800/50 pt-4 mt-4">
        <Button label="Rerun" />
      </div>
      <div className="pr-4 mt-4">
        {timeline?.map((row, i, list) => (
          <FunctionRunTimelineRow
            createdAt={row.createdAt}
            rowType={row.__typename === 'FunctionEvent' ? 'function' : 'step'}
            eventType={
              row.__typename === 'FunctionEvent'
                ? row.functionType || FunctionEventType.Completed
                : row.stepType || StepEventType.Completed
            }
            output={row.output}
            name={row.__typename === 'StepEvent' ? row.name || undefined : undefined}
            last={i === list.length - 1}
          />
        ))}
      </div>
    </ContentCard>
  );
};

type FunctionRunTimelineRowProps = {
  rowType: 'function' | 'step';
  eventType: FunctionEventType | StepEventType;
  output: string | null | undefined;
  createdAt: string | number;
  name?: string;
  last?: boolean;
};

const FunctionRunTimelineRow = ({
  rowType,
  eventType,
  output,
  createdAt,
  name,
  last,
}: FunctionRunTimelineRowProps) => {
  const json = useMemo(() => {
    try {
      return JSON.parse(output || '');
    } catch (e) {
      return null;
    }
  }, [output]);
  const payload = usePrettyJson(output);

  const { label, status } = useMemo(() => {
    if (rowType === 'function') {
      return functionEventTypeMap[eventType];
    }

    const stepData = stepEventTypeMap[eventType as StepEventType];

    const prefix =
      !name || name === 'step' ? 'Step' : name === '$trigger' ? 'First call' : `Step "${name}"`;

    let suffix = '';

    // If we're waiting, check how long we're waiting for.
    if (stepData.status === EventStatus.Paused) {
      try {
        if (typeof output === 'string') {
          // We're waiting for a date.
          const date = new Date(output).valueOf();
          const diff = date - new Date(createdAt).valueOf();
          suffix = `for ${ms(diff, { long: true })}`;
        }
      } catch (e) {}
    }

    return {
      ...stepData,
      label: `${prefix} ${stepData.label} ${suffix}`.trim(),
    };
  }, [rowType, eventType, name]);

  const tabs = [{ label: 'Output', content: payload || '' }];

  // sdkError stores the error contents of the SDK response.
  const sdkError = !!json?.error && (json?.output?.body || json?.output);
  if (!!sdkError) {
    tabs.push({ label: 'Error', content: sdkError });
  }

  return (
    <TimelineRow status={status} iconOffset={0} bottomLine={!last}>
      <TimelineFuncProgress label={label} date={createdAt} id="">
        {payload ? <CodeBlock tabs={tabs} /> : null}
      </TimelineFuncProgress>
    </TimelineRow>
  );
};

const functionEventTypeMap: Record<
  FunctionEventType,
  { status: EventStatus | FunctionRunStatus; label: string }
> = {
  [FunctionEventType.Cancelled]: {
    label: 'Function Cancelled',
    status: FunctionRunStatus.Cancelled,
  },
  [FunctionEventType.Completed]: {
    label: 'Function Completed',
    status: FunctionRunStatus.Completed,
  },
  [FunctionEventType.Failed]: {
    label: 'Function Failed',
    status: EventStatus.Failed,
  },
  [FunctionEventType.Started]: {
    label: 'Function Started',
    status: EventStatus.Completed,
  },
};

const stepEventTypeMap: Record<
  StepEventType,
  { status: EventStatus | FunctionRunStatus; label: string }
> = {
  [StepEventType.Completed]: {
    label: 'ran',
    status: EventStatus.Completed,
  },
  [StepEventType.Failed]: { label: 'Step Failed', status: EventStatus.Failed },
  [StepEventType.Started]: {
    label: 'started',
    status: EventStatus.Completed,
  },
  [StepEventType.Errored]: {
    label: 'errored',
    status: EventStatus.Failed,
  },
  [StepEventType.Scheduled]: {
    label: 'scheduled',
    status: EventStatus.Completed,
  },
  [StepEventType.Waiting]: {
    label: 'waiting',
    status: EventStatus.Paused,
  },
};

// TODO: Normalize this type in generated.ts
type Timeline = null | Array<
  | {
      __typename: 'FunctionEvent';
      createdAt?: any | null;
      output?: string | null;
      functionType?: FunctionEventType | null;
    }
  | {
      __typename: 'StepEvent';
      createdAt?: any | null;
      output?: string | null;
      name?: string | null;
      stepType?: StepEventType | null;
      waitingFor?: {
        __typename?: 'StepEventWait';
        expiryTime: any;
        eventName?: string | null;
        expression?: string | null;
      } | null;
    }
>;

const normalizeSteps = (timeline: Timeline): Timeline => {
  // Normalize the feed here.  The dev server API gives us _every_ event;
  // if a step is scheduled then runs immediately we can hide the scheduled
  // event.  Similarly, if a step starts then finishes immediately we can show
  // only the "Step finished" event.
  if (!timeline) return [];

  // TODO: When we include job IDs in history entries we can filter the timeline
  // by job ID:  if we have a "started" item for a job ID, don't show scheduled:
  // just show started + latency as the history info.
  const filtered = timeline.map((item, n) => {
    if (item.__typename === 'FunctionEvent') return item;

    switch (item.stepType) {
      // Clean up scheduled and started.
      case StepEventType.Scheduled: {
        // If the scheduled at time is different to the historical time,
        // show the timestamp.
        const output = JSON.parse(item.output || 'null');
        const diff = new Date(output).valueOf() - new Date(item.createdAt).valueOf();

        if (output && diff > 999) {
          // Only show "waiting" if we're waiting for longer than a second.  Naturally,
          // there may be milliseconds of difference right now adding a history log
          // and something to the queue, depending on the backend implementation.
          return { ...item, stepType: StepEventType.Waiting, output };
        }

        const next = timeline[n + 1];
        if (!next || next.__typename === 'FunctionEvent') return item;

        // Don't show this if the next step is started.
        if (next.stepType === 'STARTED') return null;
      }
      case StepEventType.Started: {
        const next = timeline[n + 1];
        if (!next || next.__typename === 'FunctionEvent') return item;
        // Don't show this if the next step is completed.
        if (next.stepType === 'COMPLETED' || next.stepType === 'FAILED') return null;
      }
    }
    return item;
  });

  return filtered.filter(Boolean) as Timeline;
};
