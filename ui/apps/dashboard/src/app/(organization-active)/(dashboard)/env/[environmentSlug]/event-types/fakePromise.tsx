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
            functions: [
              {
                id: '1e8f5106-b6f6-4ed7-9adf-da4e48335038',
                slug: 'growth-product-analytics-account-created',
                name: 'Product Analytics: Account Created',
              },
              {
                id: '641bb1d0-1c58-402c-a86a-62baee698f2c',
                slug: 'growth-slack-new-account-notification',
                name: 'Slack: New account notification',
              },
            ],
            // TODO: This will be another query, lazy loaded
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
