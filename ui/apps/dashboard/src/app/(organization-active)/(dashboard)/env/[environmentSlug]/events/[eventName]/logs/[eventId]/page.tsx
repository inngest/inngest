import { notFound } from 'next/navigation';
import { Time } from '@inngest/components/Time';

import SkippedFunctionCard from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/events/[eventName]/logs/[eventId]/SkippedFunctionCard';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import EventPayload from './EventPayload';
import TriggeredFunctionCard from './TriggeredFunctionCard';

const GetEventDocument = graphql(`
  query GetEvent($environmentID: ID!, $eventID: ULID!) {
    environment: workspace(id: $environmentID) {
      event: archivedEvent(id: $eventID) {
        receivedAt
        ...EventPayload
        functionRuns {
          id
          function {
            id
          }
        }
        skippedFunctionRuns {
          id
          skipReason
          workflowID
          skippedAt
        }
      }
    }
  }
`);

type EventPageProps = {
  params: {
    environmentSlug: string;
    eventName: string;
    eventId: string;
  };
};

export const runtime = 'nodejs';

export default async function EventPage({ params }: EventPageProps) {
  const environment = await getEnvironment({ environmentSlug: params.environmentSlug });
  const response = await graphqlAPI.request(GetEventDocument, {
    environmentID: environment.id,
    eventID: params.eventId,
  });

  const event = response.environment.event;

  if (!event) {
    notFound();
  }

  return (
    <div className="flex h-full flex-col overflow-y-scroll">
      <header className="bg-canvasBase space-y-1 p-5 shadow">
        <h2 className="font-medium capitalize">
          <Time format="relative" value={new Date(event.receivedAt)} />
        </h2>
        <dl className="text-subtle flex justify-between text-sm">
          <dt className="sr-only">Event Timestamp</dt>
          <dl>
            <Time value={new Date(event.receivedAt)} />
          </dl>
          <dt className="sr-only">Event ID</dt>
          <dl className="font-mono">{params.eventId}</dl>
        </dl>
      </header>
      <main className="flex min-h-0 flex-1">
        <div className="min-w-0 flex-1 p-5">
          <EventPayload event={event} />
        </div>
        <div className="relative">
          <div className="border-subtle absolute top-5 h-full border-r" />
        </div>
        <div className="w-2/6 flex-shrink-0 space-y-4 overflow-y-auto p-5">
          {event.skippedFunctionRuns.length > 0 && (
            <>
              <h3 className="font-medium">Skipped Functions</h3>
              <ul className="space-y-3 pb-4">
                {event.skippedFunctionRuns.map((skippedRun) => (
                  <li key={'skipped:' + skippedRun.id}>
                    <SkippedFunctionCard
                      environmentSlug={params.environmentSlug}
                      environmentID={environment.id}
                      functionID={skippedRun.workflowID}
                      skipReason={skippedRun.skipReason}
                      skippedAt={new Date(skippedRun.skippedAt)}
                    />
                  </li>
                ))}
              </ul>
            </>
          )}
          <h3 className="font-medium">Triggered Functions</h3>
          <ul className="space-y-3">
            {event.functionRuns.length === 0 ? (
              <p className="my-4 text-sm leading-6">No functions triggered by this event.</p>
            ) : (
              event.functionRuns.map((functionRun) => (
                <li key={functionRun.id}>
                  <TriggeredFunctionCard
                    environmentSlug={params.environmentSlug}
                    environmentID={environment.id}
                    functionID={functionRun.function.id}
                    functionRunID={functionRun.id}
                  />
                </li>
              ))
            )}
          </ul>
        </div>
      </main>
    </div>
  );
}
