import Link from 'next/link';
import { Time } from '@inngest/components/Time';
import { IconFunction } from '@inngest/components/icons/Function';
import { IconStatusCancelled } from '@inngest/components/icons/status/Cancelled';
import { IconStatusCompleted } from '@inngest/components/icons/status/Completed';
import { IconStatusFailed } from '@inngest/components/icons/status/Failed';
import { IconStatusPaused } from '@inngest/components/icons/status/Paused';
import { IconStatusQueued } from '@inngest/components/icons/status/Queued';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';
import { RiArrowRightSLine } from '@remixicon/react';
import { noCase } from 'change-case';
import { titleCase } from 'title-case';

import { graphql } from '@/gql';
import { FunctionRunStatus } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';

const functionRunStatusIcons: Record<string, (args: { className?: string }) => React.ReactNode> = {
  [FunctionRunStatus.Cancelled]: IconStatusCancelled,
  [FunctionRunStatus.Completed]: IconStatusCompleted,
  [FunctionRunStatus.Failed]: IconStatusFailed,
  [FunctionRunStatus.Running]: IconStatusRunning,
  [FunctionRunStatus.Queued]: IconStatusQueued,
  [FunctionRunStatus.Paused]: IconStatusPaused,
};

const GetFunctionRunCardDocument = graphql(`
  query GetFunctionRunCard($environmentID: ID!, $functionID: ID!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflow(id: $functionID) {
        name
        slug
        run(id: $functionRunID) {
          id
          status
          startedAt
        }
      }
    }
  }
`);

type TriggeredFunctionCardProps = {
  environmentSlug: string;
  environmentID: string;
  functionID: string;
  functionRunID: string;
};

export default async function TriggeredFunctionCard({
  environmentSlug,
  environmentID,
  functionID,
  functionRunID,
}: TriggeredFunctionCardProps) {
  const response = await graphqlAPI.request(GetFunctionRunCardDocument, {
    environmentID,
    functionID,
    functionRunID,
  });

  const function_ = response.environment.function;

  if (!function_) {
    return null;
  }

  const StatusIcon = functionRunStatusIcons[function_.run.status] ?? IconStatusCancelled;
  if (functionRunStatusIcons[function_.run.status] === undefined) {
    console.error(
      `[TriggeredFunctionCard] missing function run status icon: ${function_.run.status}`
    );
  }

  return (
    <Link
      href={`/env/${environmentSlug}/functions/${encodeURIComponent(function_.slug)}/logs/${
        function_.run.id
      }`}
      className="flex items-center rounded-lg border bg-white p-5 shadow"
    >
      <div className="flex-1">
        <div className="flex items-center gap-1.5">
          <IconFunction className="h-4 w-4 text-slate-500" />
          <h4 className="font-medium text-slate-800">{function_.name}</h4>
        </div>
        <dl>
          <dt className="sr-only">Triggered at</dt>
          <dd>
            <Time
              className="text-xs text-slate-500"
              format="relative"
              value={new Date(function_.run.startedAt)}
            />
          </dd>
          <dt className="sr-only">Status</dt>
          <dd className="mt-2 flex items-center gap-1.5 font-medium text-slate-500">
            <StatusIcon className="h-4 w-4" /> {titleCase(noCase(function_.run.status))}
          </dd>
        </dl>
      </div>
      <div className="shrink-0">
        <RiArrowRightSLine className="h-5 w-5 text-slate-400" />
      </div>
    </Link>
  );
}
