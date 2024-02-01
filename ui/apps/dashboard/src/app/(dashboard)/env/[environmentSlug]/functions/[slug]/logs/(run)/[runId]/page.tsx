import { notFound } from 'next/navigation';

import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import { StreamDetails } from './StreamDetails';

const GetFunctionRunDetailsDocument = graphql(`
  query GetFunctionRunDetails($environmentID: ID!, $functionSlug: String!, $functionRunID: ULID!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        name
        run(id: $functionRunID) {
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

export default async function FunctionRunDetailsLayout({ params }: FunctionRunDetailsLayoutProps) {
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

  const func = response.environment.function;
  const triggers = (run.version?.triggers ?? []).map((trigger) => {
    return {
      type: trigger.schedule ? 'CRON' : 'EVENT',
      value: trigger.schedule ?? trigger.eventName ?? '',
    } as const;
  });

  const events = func?.run?.events.map((event) => {
    return {
      ...event,
      receivedAt: new Date(event.receivedAt),
    };
  });

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
