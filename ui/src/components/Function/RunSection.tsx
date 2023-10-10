import { useCallback, useEffect, useMemo, useState } from 'react';
import ms from 'ms';

import { type OutputType } from '@/components/Function/OutputRenderer';
import MetadataGrid from '@/components/Metadata/MetadataGrid';
import { IconClock } from '@/icons';
import { client } from '@/store/baseApi';
import { usePrettyJson } from '../../hooks/usePrettyJson';
import {
  EventStatus,
  FunctionEventType,
  FunctionRunStatus,
  FunctionTriggerTypes,
  GetHistoryItemOutputDocument,
  StepEventType,
  useGetFunctionRunQuery,
} from '../../store/generated';
import Badge from '../Badge';
import { BlankSlate } from '../Blank';
import CodeBlock from '../Code/CodeBlock';
import ContentCard from '../Content/ContentCard';
import TimelineFuncProgress from '../Timeline/TimelineFuncProgress';
import TimelineRow from '../Timeline/TimelineRow';
import { Timeline } from '../TimelineV2';
import { useParsedHistory } from '../TimelineV2/historyParser';
import OutputCard from './Output';
import renderRunMetadata from './RunMetadataRenderer';
import { FunctionRunStatusIcons } from './RunStatusIcons';
import { SleepingSummary } from './SleepingSummary';
import { WaitingSummary } from './WaitingSummary';

// TODO: Delete this. It's only here to make it easy to switch between the old and new timeline during dev.
const isNewTimelineVisible = false;

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
  const history = useParsedHistory(run?.history ?? []);
  const firstTrigger = run?.function?.triggers?.[0] ?? null;
  const cron = firstTrigger && firstTrigger.type === FunctionTriggerTypes.Cron;

  useEffect(() => {
    if (!run?.event?.id) {
      return;
    }
  }, [run?.event?.id]);

  const getOutput = useCallback(
    (historyItemID: string) => {
      if (!runId) {
        // Should be unreachable.
        return new Promise<string>((resolve) => resolve(''));
      }

      return getHistoryItemOutput({ historyItemID, runID: runId });
    },
    [runId],
  );

  if (query.isLoading) {
    return (
      <ContentCard>
        <div className="w-full h-full flex items-center justify-center p-8">
          <div className="opacity-75 italic">Loading...</div>
        </div>
      </ContentCard>
    );
  }

  if (!run || !runId) {
    return (
      <ContentCard>
        <BlankSlate
          imageUrl="/images/no-results.png"
          title="No functions called"
          subtitle="Read our documentation to learn how to write functions"
          link={{
            text: 'Writing Functions',
            url: 'https://www.inngest.com/docs/functions',
          }}
        />
      </ContentCard>
    );
  }
  const metadataItems = renderRunMetadata(run);
  let type: OutputType | undefined;
  if (run.status === FunctionRunStatus.Completed) {
    type = 'completed';
  } else if (run.status === FunctionRunStatus.Failed) {
    type = 'failed';
  }

  return (
    <ContentCard
      title={run.name || 'Unknown'}
      icon={run.status && <FunctionRunStatusIcons status={run.status} className="icon-xl" />}
      type="run"
      badge={
        cron ? (
          <div className="py-2">
            <Badge className="text-orange-400 bg-orange-400/10" kind="solid">
              <IconClock />
              {firstTrigger.value}
            </Badge>
          </div>
        ) : null
      }
      metadata={
        <div className="pt-8">
          <MetadataGrid metadataItems={metadataItems} />
        </div>
      }
    >
      <div className="px-5 pt-4">
        {run.status && run.finishedAt && run.output && type && (
          <OutputCard content={run.output} type={type} />
        )}

        <WaitingSummary history={history} />
        <SleepingSummary history={history} />
      </div>

      <hr className="border-slate-800/50 mt-8" />
      <div className="px-5 pt-4">
        <h3 className="text-slate-400 text-sm py-4">Timeline</h3>
        {isNewTimelineVisible && <Timeline getOutput={getOutput} history={history} />}

        {!isNewTimelineVisible &&
          timeline?.map((row, i, list) => (
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

async function getHistoryItemOutput({
  historyItemID,
  runID,
}: {
  historyItemID: string;
  runID: string;
}): Promise<string> {
  // TODO: How to get type annotations? It returns `any`.
  const res: unknown = await client.request(GetHistoryItemOutputDocument, {
    historyItemID,
    runID,
  });

  if (typeof res !== 'object' || res === null || !('functionRun' in res)) {
    throw new Error('invalid response');
  }
  const { functionRun } = res;

  if (
    typeof functionRun !== 'object' ||
    functionRun === null ||
    !('historyItemOutput' in functionRun)
  ) {
    throw new Error('invalid response');
  }
  const { historyItemOutput } = functionRun;

  if (typeof historyItemOutput !== 'string') {
    throw new Error('invalid response');
  }

  return historyItemOutput;
}
