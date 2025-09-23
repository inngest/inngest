import { useRouter } from 'next/navigation';
import { RiLightbulbLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { Button } from '../Button';
import { ErrorCard } from '../Error/ErrorCard';
import { Pill } from '../Pill';
import { useGetDebugSession, type DebugSessionRun } from '../SharedContext/useGetDebugSession';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { Skeleton } from '../Skeleton';
import { StatusCell, Table, TimeCell } from '../Table';
import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';

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

type HistoryTable = DebugSessionRun | null;

export const History = ({ functionSlug, debugSessionID, runID }: HistoryProps) => {
  const { pathCreator } = usePathCreator();
  const router = useRouter();
  const { data, loading, error } = useGetDebugSession({
    functionSlug,
    debugSessionID,
    runID,
    refetchInterval: 1000,
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

  const load = (debugRunID: string) => {
    const debuggerPath = pathCreator.debugger({
      functionSlug,
      runID,
      debugRunID,
      debugSessionID,
    });

    router.push(debuggerPath);
  };

  const columnHelper = createColumnHelper<HistoryTable>();

  const columns = [
    columnHelper.accessor('startedAt', {
      cell: (rawStartedAt) => {
        const startedAt = rawStartedAt.getValue();
        return <TimeCell date={startedAt ? new Date(startedAt) : '--'} />;
      },
      size: 25,
      enableSorting: true,
    }),
    columnHelper.accessor('status', {
      cell: (rawStatus) => {
        const status = rawStatus.getValue();
        return <StatusCell key={status} status={status} label={status} size="small" />;
      },
      enableSorting: false,
    }),
    columnHelper.accessor('tags', {
      cell: () => {
        return (
          <Tooltip>
            <TooltipTrigger>
              <Pill appearance="outlined" kind="primary">
                <div
                  className="flex flex-row items-center gap-1"
                  onClick={(e) => {
                    e.stopPropagation();
                  }}
                >
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

  if (data.debugRuns?.length === 0) {
    return null;
  }

  return (
    <div className="flex w-full flex-col justify-start gap-2">
      <Table
        noHeader={true}
        onRowClick={(row) =>
          row.original &&
          router.push(
            pathCreator.debugger({
              functionSlug,
              runID,
              debugSessionID: runID,
              debugRunID: row.original.debugRunID,
            })
          )
        }
        data={(data.debugRuns ?? []).sort(
          (a, b) =>
            (b?.startedAt ? new Date(b.startedAt).getTime() : 0) -
            (a?.startedAt ? new Date(a.startedAt).getTime() : 0)
        )}
        columns={columns}
      />
    </div>
  );
};
