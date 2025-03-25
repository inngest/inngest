import React from 'react';
import NextLink from 'next/link';
import { Time } from '@inngest/components/Time';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { IconStatusCancelled } from '@inngest/components/icons/status/Cancelled';
import { IconStatusPaused } from '@inngest/components/icons/status/Paused';
import { RiArrowRightSLine } from '@remixicon/react';

import { graphql } from '@/gql';
import { SkipReason } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';

const skipReasonStatusIcons: Record<string, (args: { className?: string }) => React.ReactNode> = {
  [SkipReason.None]: IconStatusCancelled,
  [SkipReason.FunctionPaused]: IconStatusPaused,
};

const skipReasonDescriptions: Record<string, string> = {
  [SkipReason.None]: 'Skipped',
  [SkipReason.FunctionPaused]: 'Function was paused',
};

const GetFunctionNameSlugDocument = graphql(`
  query GetFunctionNameSlug($environmentID: ID!, $functionID: ID!) {
    environment: workspace(id: $environmentID) {
      function: workflow(id: $functionID) {
        name
        slug
      }
    }
  }
`);

type SkippedFunctionCardProps = {
  environmentSlug: string;
  environmentID: string;
  functionID: string;
  skipReason: SkipReason;
  skippedAt: Date;
};

export default async function SkippedFunctionCard({
  environmentSlug,
  environmentID,
  functionID,
  skipReason,
  skippedAt,
}: SkippedFunctionCardProps) {
  const response = await graphqlAPI.request(GetFunctionNameSlugDocument, {
    environmentID,
    functionID,
  });
  const function_ = response.environment.function;
  if (!function_) {
    return null;
  }

  const StatusIcon = skipReasonStatusIcons[skipReason] ?? IconStatusCancelled;

  if (skipReasonStatusIcons[skipReason] === undefined) {
    console.error(`[SkippedFunctionCard] missing skip reason icon: ${skipReason}`);
  }
  if (skipReasonDescriptions[skipReason] === undefined) {
    console.error(`[SkippedFunctionCard] missing skip reason description: ${skipReason}`);
  }

  return (
    <NextLink
      href={`/env/${environmentSlug}/functions/${encodeURIComponent(function_.slug)}`}
      className="bg-canvasBase flex items-center rounded-md border p-5 shadow"
    >
      <div className="flex-1">
        <div className="flex items-center gap-1.5">
          <FunctionsIcon className="text-subtle h-4 w-4" />
          <h4 className="font-medium">{function_.name}</h4>
        </div>
        <dl>
          <dt className="sr-only">Skipped at</dt>
          <dd>
            <Time className="text-subtle text-xs" format="relative" value={skippedAt} />
          </dd>
          <dt className="sr-only">Status</dt>
          <dd className="text-subtle mt-2 flex items-center gap-1.5 font-medium">
            <StatusIcon className="h-4 w-4" /> {skipReasonDescriptions[skipReason]}
          </dd>
        </dl>
      </div>
      <div className="shrink-0">
        <RiArrowRightSLine className="text-subtle h-5 w-5" />
      </div>
    </NextLink>
  );
}
