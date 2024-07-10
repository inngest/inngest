'use client';

import { notFound } from 'next/navigation';
import { EventDetails } from '@inngest/components/EventDetails';
import { RunDetails } from '@inngest/components/RunDetails';

import { useEnvironment } from '@/components/Environments/EnvContext';
import { graphql } from '@/gql';
import cn from '@/utils/cn';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { StreamDetails } from './StreamDetails';

const GetFunctionRunDetailsDocument = graphql(`
  query GetFunctionRunDetails($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        name
        run(id: $functionRunID) {
          batchID
          canRerun
          events {
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
            invokeFunction {
              eventID
              functionID
              correlationID
              timeout
            }
            invokeFunctionResult {
              eventID
              timeout
              runID
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
};

export default function FunctionRunDetailsLayout({ params }: FunctionRunDetailsLayoutProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const environment = useEnvironment();

  const res = useGraphQLQuery({
    query: GetFunctionRunDetailsDocument,
    variables: {
      environmentID: environment.id,
      functionSlug,
      functionRunID: params.runId,
    },
  });

  if (res.error) {
    throw res.error;
  }
  if (res.isLoading) {
    return (
      <div className={cn('dark grid h-full text-white', 'grid-cols-2')}>
        <EventDetails loading />
        <RunDetails loading />
      </div>
    );
  }

  const { run } = res.data.environment.function ?? {};

  if (!run) {
    notFound();
  }

  const func = res.data.environment.function;
  const triggers = (run.version?.triggers ?? []).map((trigger) => {
    return {
      type: trigger.schedule ? 'CRON' : 'EVENT',
      value: trigger.schedule ?? trigger.eventName ?? '',
    } as const;
  });

  const events = func?.run.events
    ? func.run.events.map((event) => {
        return {
          ...event,
          receivedAt: new Date(event.receivedAt),
        };
      })
    : undefined;

  if (!func) {
    throw new Error('missing function');
  }

  return (
    <StreamDetails
      environment={{
        id: environment.id,
        slug: params.environmentSlug,
      }}
      events={events}
      func={{
        ...func,
        triggers,
      }}
      functionVersion={run.version ?? undefined}
      rawHistory={run.history}
      run={{
        ...run,
        endedAt: run.endedAt ? new Date(run.endedAt) : null,
        output: run.output ?? null,
        startedAt: new Date(run.startedAt),
      }}
    />
  );
}
