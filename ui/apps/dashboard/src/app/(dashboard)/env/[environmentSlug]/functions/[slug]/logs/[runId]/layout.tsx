import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ClockIcon, RectangleStackIcon, RocketLaunchIcon } from '@heroicons/react/20/solid';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import EventIcon from '@/icons/event.svg';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import FunctionRunStatusCard from './FunctionRunStatusCard';
import { StreamDetails } from './StreamDetails';

const GetFunctionRunDetailsDocument = graphql(`
  query GetFunctionRunDetails($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        name
        run(id: $functionRunID) {
          canRerun
          event {
            id
            name
            payload: event
            receivedAt
          }
          history {
            attempt
            cancel {
              eventID
              expression
              userID
            }
            createdAt
            functionVersion
            groupID
            id
            sleep {
              until
            }
            stepName
            type
            url
            waitForEvent {
              eventName
              expression
              timeout
            }
            waitResult {
              eventID
              timeout
            }
          }
          id
          status
          startedAt
          endedAt
          output
          version: workflowVersion {
            deploy {
              id
              createdAt
            }
            triggers {
              eventName
              schedule
            }
            url
            validFrom
            version
          }
        }
        slug
      }
    }
  }
`);

type FunctionRunDetailsLayoutProps = {
  params: {
    environmentSlug: string;
    slug: string;
    runId: string;
  };
  children: React.ReactNode;
};

export default async function FunctionRunDetailsLayout({
  params,
  children,
}: FunctionRunDetailsLayoutProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const environment = await getEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const response = await graphqlAPI.request(GetFunctionRunDetailsDocument, {
    environmentID: environment.id,
    functionSlug,
    functionRunID: params.runId,
  });

  const { run } = response.environment.function ?? {};

  if (!run) {
    notFound();
  }

  const eventName = run.version.triggers[0]?.eventName;
  const scheduleName = run.version.triggers[0]?.schedule;
  const func = response.environment?.function;
  const triggers = (run?.version?.triggers ?? []).map((trigger) => {
    return {
      type: trigger.schedule ? 'CRON' : 'EVENT',
      value: trigger.schedule ?? trigger.eventName ?? '',
    } as const;
  });
  const { event } = func?.run ?? {};

  if (!func) {
    throw new Error('missing function');
  }

  if (await getBooleanFlag('new-run-details')) {
    return (
      <StreamDetails
        environment={{
          id: environment.id,
          slug: params.environmentSlug,
        }}
        event={
          event
            ? {
                ...event,
                receivedAt: new Date(event.receivedAt),
              }
            : undefined
        }
        func={{
          ...func,
          triggers,
        }}
        functionVersion={run.version ?? undefined}
        rawHistory={run?.history ?? []}
        run={{
          ...run,
          endedAt: run.endedAt ?? null,
          output: run.output ?? null,
        }}
      />
    );
  }

  // Everything below this line is the legacy details.

  let triggerCard: React.ReactNode;
  if (eventName) {
    triggerCard = (
      <Link
        href={`/env/${params.environmentSlug}/events/${encodeURIComponent(eventName)}`}
        className="block space-y-1 overflow-hidden rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow hover:bg-slate-50"
      >
        <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
          <EventIcon className="h-3 w-3 text-indigo-500" />
          {eventName}
        </div>
        <div className="text-sm text-slate-500">event</div>
      </Link>
    );
  } else if (scheduleName) {
    triggerCard = (
      <div className="space-y-1 overflow-hidden rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow">
        <div className="flex items-center gap-2 text-sm text-slate-900">
          <ClockIcon className="h-3 w-3 text-indigo-500" />
          {scheduleName}
        </div>
        <div className="text-sm text-slate-500">schedule</div>
      </div>
    );
  }

  return (
    <div className="flex h-full">
      <main className="w-96 flex-shrink-0 overflow-y-auto px-5 py-4">
        <FunctionRunStatusCard status={run.status} />
        <header className="mt-6 flex flex-col gap-1 ">
          <h1 className="font-medium text-slate-800">
            Started at: <Time value={new Date(run.startedAt)} />
          </h1>

          {run.endedAt && (
            <h1 className="font-medium text-slate-800">
              Ended at: <Time value={new Date(run.endedAt)} />
            </h1>
          )}
          <span className="font-mono text-xs text-slate-400">Run ID: {run.id}</span>
        </header>
        <div className="mt-6 space-y-3">
          <Link
            href={`/env/${params.environmentSlug}/functions/${params.slug}/versions`}
            className="block overflow-hidden rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow hover:bg-slate-50"
          >
            <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
              <RectangleStackIcon className="h-3 w-3 text-sky-500" />
              {`version-${run.version.version}`}
            </div>

            {run.version.validFrom && (
              <Time
                className="text-sm text-slate-500"
                format="relative"
                value={new Date(run.version.validFrom)}
              />
            )}
          </Link>
          {run.version.deploy && (
            <Link
              href={`/env/${params.environmentSlug}/deploys/${run.version.deploy.id}`}
              className="block overflow-hidden rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow hover:bg-slate-50"
            >
              <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
                <RocketLaunchIcon className="h-3 w-3 text-indigo-500" />
                {run.version.deploy.id}
              </div>
              <span className="text-sm text-slate-500">
                Deployed <Time format="relative" value={new Date(run.version.deploy.createdAt)} />
              </span>
            </Link>
          )}
          {triggerCard}

          <div className="block overflow-scroll rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow">
            <div className="flex items-center gap-2 whitespace-nowrap text-sm font-medium text-slate-900">
              {run.version.url}
            </div>
            <span className="text-sm text-slate-500">URL</span>
          </div>
        </div>
      </main>
      <aside className="h-full min-w-0 flex-1 py-4 pr-4">{children}</aside>
    </div>
  );
}
