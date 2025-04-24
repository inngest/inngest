import { useCallback } from 'react';

type QueryVariables = {
  eventName?: string[];
  cursor?: string | null;
  source?: string;
  startTime?: string;
  celQuery?: string;
};
// TODO: Replace with real API
export function useEvents() {
  return useCallback(async ({ cursor, eventName, source, startTime, celQuery }: QueryVariables) => {
    console.log(cursor, eventName, source, startTime, celQuery);
    // Simulated delay to mimic real API behavior
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Example mocked data
    const events = [
      {
        id: '01JGPM6FYSRN9C0ZGJ7PXPVRGY',
        receivedAt: new Date('2025-04-10T16:43:21.696Z'),
        name: 'UserSignedUp',
        runs: [
          {
            fnName: 'SendWelcomeEmail',
            fnSlug: 'send-welcome-email',
            status: 'COMPLETED',
            id: 'id-1',
            startedAt: new Date('2025-04-10T16:43:22.696Z'),
            completedAt: new Date('2025-04-10T16:43:24.696Z'),
          },
          {
            fnName: 'SendNewsletter',
            fnSlug: 'send-newsletter',
            status: 'CANCELLED',
            id: 'id-2',
            startedAt: new Date('2025-04-10T16:43:23.696Z'),
            completedAt: new Date('2025-04-10T16:43:24.696Z'),
          },
        ],
      },
    ];

    return {
      events,
      pageInfo: {
        hasNextPage: false,
        hasPreviousPage: false,
        endCursor: null,
        startCursor: null,
      },
      totalCount: 1,
    };
  }, []);
}

// TODO: Replace with real API
export function useEventDetails() {
  return useCallback(async ({ eventName }: { eventName: string }) => {
    console.log(eventName);
    await new Promise((resolve) => setTimeout(resolve, 500));

    const event = {
      id: '01JGPM6FYSRN9C0ZGJ7PXPVRGY',
      receivedAt: new Date('2025-04-10T16:43:21.696Z'),
      idempotencyKey: 'custom-payload-id',
      name: 'UserSignedUp',
      source: 'Default Inngest key',
      timestamp: new Date(1745226902417),
      version: '2022-12-16',
    };

    return event;
  }, []);
}

export function useEventPayload() {
  return useCallback(async ({ eventName }: { eventName: string }) => {
    console.log(eventName);
    await new Promise((resolve) => setTimeout(resolve, 500));

    const event = {
      name: 'UserSignedUp',
      payload:
        '{\n  "name": "signup.new",\n  "data": {\n    "account_id": "119f5971-9878-46bd-a18f-4fecd",\n    "method": "",\n    "plan_name": "Free Tier"\n  },\n  "id": "119f5971-9878-46bd-a18f-4f0680174ecd",\n  "ts": 1711051784369,\n  "v": "2021-05-11.01"\n}',
    };

    return event;
  }, []);
}
