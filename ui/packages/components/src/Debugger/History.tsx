import { RiLightbulbLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { Button } from '../Button';
import { ErrorCard } from '../Error/ErrorCard';
import { Pill } from '../Pill';
import type { RunTraceSpan } from '../SharedContext/useGetDebugRun';
import { useGetDebugSession } from '../SharedContext/useGetDebugSession';
import { Skeleton } from '../Skeleton';
import { StatusDot } from '../Status/StatusDot';
import { getStatusTextClass } from '../Status/statusClasses';
import { Table } from '../Table';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';
import { cn } from '../utils/classNames';
import { mediumDateFormat } from '../utils/date';

export const exampleAiOutput = {
  id: 'chatcmpl-BjpG3gipnAUHsi3txqSt5XLp9G76J',
  object: 'chat.completion',
  created: 1750261495,
  model: 'chatgpt-4o-latest',
  choices: [
    {
      index: 0,
      message: {
        role: 'assistant',
        content:
          "Sure! Here's a famous quote that relates to sidewalks and rain:\n\n“I always like walking in the rain, so no one can see me crying.”  \n— Charlie Chaplin\n\nWhile it doesn't mention sidewalks directly, it evokes the imagery of walking—often on sidewalks—during the rain, and expresses a poignant emotional undertone.",
        refusal: null,
        annotations: [],
      },
      logprobs: null,
      finish_reason: 'stop',
    },
  ],
  usage: {
    prompt_tokens: 17,
    completion_tokens: 65,
    total_tokens: 82,
    prompt_tokens_details: {
      cached_tokens: 0,
      audio_tokens: 0,
    },
    completion_tokens_details: {
      reasoning_tokens: 0,
      audio_tokens: 0,
      accepted_prediction_tokens: 0,
      rejected_prediction_tokens: 0,
    },
  },
  service_tier: 'default',
  system_fingerprint: 'fp_afccf7958a',
};

export const exampleInput = {
  messages: [
    {
      content: 'Give me a famous quote about sidewalks and rain.',
      role: 'user',
    },
  ],
  model: 'chatgpt-4o-latest',
  temperature: 0.9,
};

export const exampleOutput = {
  data: {
    id: 'chatcmpl-BjpG3gipnAUHsi3txqSt5XLp9G76J',
    object: 'chat.completion',
    created: 1750261495,
    model: 'chatgpt-4o-latest',
    choices: [
      {
        index: 0,
        message: {
          role: 'assistant',
          content:
            "Sure! Here's a famous quote that relates to sidewalks and rain:\n\n“I always like walking in the rain, so no one can see me crying.”  \n— Charlie Chaplin\n\nWhile it doesn't mention sidewalks directly, it evokes the imagery of walking—often on sidewalks—during the rain, and expresses a poignant emotional undertone.",
          refusal: null,
          annotations: [],
        },
        logprobs: null,
        finish_reason: 'stop',
      },
    ],
    usage: {
      prompt_tokens: 17,
      completion_tokens: 65,
      total_tokens: 82,
      prompt_tokens_details: {
        cached_tokens: 0,
        audio_tokens: 0,
      },
      completion_tokens_details: {
        reasoning_tokens: 0,
        audio_tokens: 0,
        accepted_prediction_tokens: 0,
        rejected_prediction_tokens: 0,
      },
    },
    service_tier: 'default',
    system_fingerprint: 'fp_afccf7958a',
  },
};

const data = [
  {
    id: '1',
    dateStarted: new Date('2025-06-14T12:07:00Z'),
    status: 'COMPLETED',
    tagCount: 1,
    input: exampleInput,
    output: exampleOutput,
    aiOutput: exampleAiOutput,
  },
  {
    id: '2',
    dateStarted: new Date('2025-06-12T08:12:00Z'),
    status: 'FAILED',
    tagCount: 6,
    input: exampleInput,
    output: exampleOutput,
    aiOutput: exampleAiOutput,
  },
  {
    id: '3',
    dateStarted: new Date('2025-06-11T12:32:00Z'),
    status: 'RUNNING',
    tagCount: 20,
    input: exampleInput,
    output: exampleOutput,
    aiOutput: exampleAiOutput,
  },
];

type HistoryProps = {
  functionSlug: string;
  debugSessionID?: string;
  runID?: string;
};

type HistoryTable = RunTraceSpan & {
  // TODO: remove these once we have the actual data
  tags?: string[];
  versions?: string[];
};

export const History = ({ functionSlug, debugSessionID, runID }: HistoryProps) => {
  const { data, loading, error } = useGetDebugSession({
    functionSlug,
    debugSessionID,
    runID,
  });

  if (loading) {
    return (
      <div className="flex w-full flex-col gap-2">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  if (error || !data) {
    return <ErrorCard error={error || new Error('No data found')} />;
  }

  const columnHelper = createColumnHelper<HistoryTable>();

  const columns = [
    columnHelper.accessor('startedAt', {
      cell: (rawStartedAt) => {
        const startedAt = rawStartedAt.getValue();
        return (
          <span className="text-muted text-sm leading-tight">
            {startedAt ? new Date(startedAt).toLocaleString('en-US', mediumDateFormat) : '—'}
          </span>
        );
      },
      size: 25,
      enableSorting: true,
    }),
    columnHelper.accessor('status', {
      cell: (rawStatus) => {
        const status = rawStatus.getValue();

        return (
          <div
            className={cn(
              'no-wrap flex flex-row items-center gap-2 text-sm',
              getStatusTextClass(status)
            )}
          >
            <StatusDot status={status} className="h-2.5 w-2.5 shrink-0" />
            {status}
          </div>
        );
      },
      enableSorting: false,
    }),
    columnHelper.accessor('tags', {
      cell: () => {
        return (
          <Tooltip>
            <TooltipTrigger>
              <Pill appearance="outlined" kind="primary">
                <div className="flex flex-row items-center gap-1">
                  <RiLightbulbLine className="text-muted h-2.5 w-2.5" />

                  {0}
                </div>
              </Pill>
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line text-left">
              Tags coming soon!
            </TooltipContent>
          </Tooltip>
        );
      },
      enableSorting: false,
    }),
    columnHelper.accessor('versions', {
      cell: () => {
        return (
          <Tooltip>
            <TooltipTrigger>
              <Button
                disabled={true}
                kind="secondary"
                appearance="outlined"
                size="small"
                label="View version"
                className="text-muted text-xs"
                onClick={(e) => {
                  e.stopPropagation();
                }}
              />
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line text-left">
              Version history coming soon!
            </TooltipContent>
          </Tooltip>
        );
      },
      enableSorting: false,
    }),
  ];

  return (
    <div className="flex w-full flex-col justify-start gap-2 ">
      <Table
        noHeader={true}
        data={
          data
            ? data
                .filter((run): run is RunTraceSpan => !!run && !!run.runID)
                .map((run) => ({ ...run, tags: ['tag'], versions: ['version'] }))
            : []
        }
        columns={columns}
      />
      {/* {data.map(
        (run, i) =>
          run && (
            <StepHistory
              debugRun={run}
              defaultOpen={i === data.length - 1}
              key={`step-history-${run.spanID}`}
            />
          )
      )} */}
    </div>
  );
};
