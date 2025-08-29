import { useEffect, useState } from 'react';

import { useGetDebugSession } from '../SharedContext/useGetDebugSession';
import { Skeleton } from '../Skeleton';
import { StepHistory } from './StepHistory';

const exampleAiOutput = {
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

export const History = ({ functionSlug, debugSessionID, runID }: HistoryProps) => {
  const {
    data: newData,
    loading: newLoading,
    error,
  } = useGetDebugSession({
    functionSlug,
    debugSessionID,
    runID,
  });
  const [debugSessionData, setDebugSessionData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  if (loading) {
    return (
      <div className="flex w-full flex-col gap-2">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  return (
    <div className="flex w-full flex-col gap-2">
      {data.map((item, i) => (
        <StepHistory
          {...item}
          defaultOpen={i === data.length - 1}
          key={`step-history-${item.id}`}
        />
      ))}
    </div>
  );
};
