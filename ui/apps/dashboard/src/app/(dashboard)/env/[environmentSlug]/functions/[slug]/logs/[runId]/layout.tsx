import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ClockIcon, RectangleStackIcon, RocketLaunchIcon } from '@heroicons/react/20/solid';
import { HistoryParser } from '@inngest/components/utils/historyParser';

import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import EventIcon from '@/icons/event.svg';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import FunctionRunStatusCard from './FunctionRunStatusCard';
import { StreamDetails } from './StreamDetails';

// TODO: Delete this when the new stream details are ready.
const isNewStreamDetailsVisible = false;

const GetFunctionRunDetailsDocument = graphql(`
  query GetFunctionRunDetails($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        name
        run(id: $functionRunID) {
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
          functionVersion: workflowVersion {
            validFrom
            version
            deploy {
              id
              createdAt
            }
            triggers {
              eventName
              schedule
            }
          }
          output
          version: workflowVersion {
            triggers {
              eventName
              schedule
            }
            url
          }
        }
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

  const functionRun = response.environment.function?.run;

  if (!functionRun) {
    notFound();
  }

  const eventName = functionRun.version.triggers[0]?.eventName;
  const scheduleName = functionRun.version.triggers[0]?.schedule;
  const func = response.environment?.function;
  const { run } = func ?? {};
  const triggers = (run?.version?.triggers ?? []).map((trigger) => {
    return {
      type: trigger.schedule ? 'CRON' : 'EVENT',
      value: trigger.schedule ?? trigger.eventName ?? '',
    } as const;
  });
  const { event } = func?.run ?? {};
  const history = new HistoryParser(run?.history ?? []);

  if (!func) {
    throw new Error('missing function');
  }
  if (!run) {
    throw new Error('missing run');
  }

  if (isNewStreamDetailsVisible) {
    return (
      <StreamDetails
        envID={environment.id}
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
        <FunctionRunStatusCard status={functionRun.status} />
        <header className="mt-6 flex flex-col gap-1 ">
          <h1 className="font-medium text-slate-800">
            Started at: <Time value={new Date(functionRun.startedAt)} />
          </h1>

          {functionRun.endedAt && (
            <h1 className="font-medium text-slate-800">
              Ended at: <Time value={new Date(functionRun.endedAt)} />
            </h1>
          )}
          <span className="font-mono text-xs text-slate-400">Run ID: {functionRun.id}</span>
        </header>
        <div className="mt-6 space-y-3">
          <Link
            href={`/env/${params.environmentSlug}/functions/${params.slug}/versions`}
            className="block overflow-hidden rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow hover:bg-slate-50"
          >
            <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
              <RectangleStackIcon className="h-3 w-3 text-sky-500" />
              {`version-${functionRun.functionVersion.version}`}
            </div>

            {functionRun.functionVersion.validFrom && (
              <Time
                className="text-sm text-slate-500"
                format="relative"
                value={new Date(functionRun.functionVersion.validFrom)}
              />
            )}
          </Link>
          {functionRun.functionVersion.deploy && (
            <Link
              href={`/env/${params.environmentSlug}/deploys/${functionRun.functionVersion.deploy.id}`}
              className="block overflow-hidden rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow hover:bg-slate-50"
            >
              <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
                <RocketLaunchIcon className="h-3 w-3 text-indigo-500" />
                {functionRun.functionVersion.deploy.id}
              </div>
              <span className="text-sm text-slate-500">
                Deployed{' '}
                <Time
                  format="relative"
                  value={new Date(functionRun.functionVersion.deploy.createdAt)}
                />
              </span>
            </Link>
          )}
          {triggerCard}

          <div className="block overflow-scroll rounded-md border border-slate-200 bg-white px-5 py-2.5 shadow">
            <div className="flex items-center gap-2 whitespace-nowrap text-sm font-medium text-slate-900">
              {functionRun.version.url}
            </div>
            <span className="text-sm text-slate-500">URL</span>
          </div>
        </div>
      </main>
      <aside className="h-full min-w-0 flex-1 py-4 pr-4">{children}</aside>
    </div>
  );
}
