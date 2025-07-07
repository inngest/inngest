import { useCallback } from 'react';

import { client } from '@/store/baseApi';
import { EVENTS_QUERY } from '../../coreapi';

type EventsQueryVariables = {
  eventNames: string[] | null;
  cursor: string | null;
  source?: string;
  startTime: string;
  endTime: string | null;
  celQuery?: string;
  includeInternalEvents?: boolean;
};

export function useEvents() {
  return useCallback(
    async ({
      cursor,
      endTime,
      source,
      eventNames,
      startTime,
      celQuery,
      includeInternalEvents,
    }: EventsQueryVariables) => {
      const result = await client.request(
        EVENTS_QUERY,
        {
          startTime,
          endTime,
          cursor,
          celQuery,
          eventNames,
          includeInternalEvents,
        },
        { requestPolicy: 'network-only' }
      );

      console.log({ result });

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.eventsV2) {
        throw new Error('no data returned');
      }

      const eventsData = result.eventsV2;
      const events = eventsData.edges.map(({ node }) => ({
        ...node,
        receivedAt: new Date(node.receivedAt),
        runs: node.runs.map((run) => ({
          fnName: run.function.name,
          fnSlug: run.function.slug,
          status: run.status,
          id: run.id,
          completedAt: run.endedAt ? new Date(run.endedAt) : undefined,
          startedAt: run.startedAt ? new Date(run.startedAt) : undefined,
        })),
      }));

      return {
        events,
        pageInfo: eventsData.pageInfo,
        totalCount: eventsData.totalCount,
      };
    },
    [client]
  );
}

export function useEventDetails() {
  return useCallback(async ({ eventID }: { eventID: string }) => {
    await new Promise((res) => setTimeout(res, 200));

    return {
      name: 'UserSignedUp',
      id: eventID,
      receivedAt: new Date(),
      idempotencyKey: 'fake-key',
      occurredAt: new Date(),
      version: 'v1',
      source: {
        name: 'auth-service',
      },
    };
  }, []);
}

export function useEventPayload() {
  return useCallback(async ({ eventID }: { eventID: string }) => {
    await new Promise((res) => setTimeout(res, 150));

    return {
      payload: JSON.stringify({ id: eventID, user: 'demo-user', action: 'signup' }),
    };
  }, []);
}

export function useEventRuns() {
  return useCallback(async ({ eventID }: { eventID: string }) => {
    await new Promise((res) => setTimeout(res, 150));

    return {
      name: 'UserSignedUp',
      runs: [
        {
          fnName: 'SendWelcomeEmail',
          fnSlug: 'send-welcome-email',
          status: 'COMPLETED',
          id: 'run_1',
          startedAt: new Date(),
          completedAt: new Date(),
        },
      ],
    };
  }, []);
}
