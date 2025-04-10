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
    // Simulated delay to mimic real API behavior
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Example mocked data
    const events = [
      {
        id: '1',
        receivedAt: '2025-04-10T16:43:21.696Z',
        name: 'UserSignedUp',
        functions: [
          {
            name: 'SendWelcomeEmail',
            slug: 'send-welcome-email',
            status: 'COMPLETED',
          },
          {
            name: 'SendNewsletter',
            slug: 'send-newsletter',
            status: 'CANCELLED',
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
    };
  }, []);
}
