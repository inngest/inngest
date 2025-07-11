import { useCallback } from 'react';

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
      // Mocked fake delay + sample response
      await new Promise((res) => setTimeout(res, 300));

      return {
        events: [
          {
            name: 'UserSignedUp',
            id: 'evt_1',
            receivedAt: new Date(),
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
          },
        ],
        pageInfo: {
          hasNextPage: false,
          endCursor: null,
          hasPreviousPage: false,
          startCursor: null,
        },
        totalCount: 1,
      };
    },
    []
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
