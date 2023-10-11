import { notFound } from 'next/navigation';

import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import { relativeTime, weekDayAndUTCTime } from '@/utils/date';
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
    <div className="flex h-full flex-col">
      <header className="space-y-1 bg-white p-5 shadow">
        <h2 className="font-medium capitalize text-slate-800">
          <Time format="relative" value={new Date(event.receivedAt)} />
        </h2>
        <dl className="flex justify-between text-sm text-slate-400">
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
          <div className="absolute top-5 h-full border-r border-slate-200" />
        </div>
        <div className="w-2/6 flex-shrink-0 space-y-4 overflow-y-auto p-5">
          <h3 className="font-medium text-slate-800">Triggered functions</h3>
          <ul className="space-y-3">
            {event.functionRuns.map((functionRun) => (
              <li key={functionRun.id}>
                <TriggeredFunctionCard
                  environmentSlug={params.environmentSlug}
                  environmentID={environment.id}
                  functionID={functionRun.function.id}
                  functionRunID={functionRun.id}
                />
              </li>
            ))}
          </ul>
        </div>
      </main>
    </div>
  );
}
