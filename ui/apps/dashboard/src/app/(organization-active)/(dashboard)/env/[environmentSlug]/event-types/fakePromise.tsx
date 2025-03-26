'use client';

import { type EventType, type PageInfo } from '@inngest/components/types/eventType';

export const fakeGetEventTypes = async ({}: {}) => {
  return new Promise<{ events: EventType[]; pageInfo: PageInfo; totalCount: number }>((resolve) => {
    setTimeout(() => {
      resolve({
        events: [
          {
            id: '1',
            name: 'User Signed Up',
            archived: false,
            functions: [],
            volume: { totalVolume: 0, chart: null },
          },
          {
            id: '2',
            name: 'Order Placed',
            archived: true,
            functions: [],
            volume: { totalVolume: 0, chart: null },
          },
        ],
        pageInfo: {
          hasNextPage: false,
          hasPreviousPage: false,
          endCursor: null,
          startCursor: null,
        },
        totalCount: 2,
      });
    }, 1000); // Simulating network delay
  });
};
