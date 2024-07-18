import React from 'react';
import Link from 'next/link';
import { Time } from '@inngest/components/Time';
import { IconFunction } from '@inngest/components/icons/Function';
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
    <Link
      href={`/env/${environmentSlug}/functions/${encodeURIComponent(function_.slug)}`}
      className="flex items-center rounded-lg border bg-white p-5 shadow"
    >
      <div className="flex-1">
        <div className="flex items-center gap-1.5">
          <IconFunction className="h-4 w-4 text-slate-500" />
          <h4 className="font-medium text-slate-800">{function_.name}</h4>
        </div>
        <dl>
          <dt className="sr-only">Skipped at</dt>
          <dd>
            <Time className="text-xs text-slate-500" format="relative" value={skippedAt} />
          </dd>
          <dt className="sr-only">Status</dt>
          <dd className="mt-2 flex items-center gap-1.5 font-medium text-slate-500">
            <StatusIcon className="h-4 w-4" /> {skipReasonDescriptions[skipReason]}
          </dd>
        </dl>
      </div>
      <div className="shrink-0">
        <RiArrowRightSLine className="h-5 w-5 text-slate-400" />
      </div>
    </Link>
  );
}
