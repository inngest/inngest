import type { InsightsFetchResult } from './types';

function getMockPage(cursor: string | null): InsightsFetchResult {
  const pageSize = 30;
  const totalCount = 100;
  const offset = cursor ? parseInt(cursor) : 0;
  const events = ['user.signup', 'user.login', 'user.logout', 'page.view'];

  const entries = Array.from({ length: pageSize }, (_, i) => {
    const index = offset + i;
    const now = new Date();
    const evenHour = new Date(
      now.getFullYear(),
      now.getMonth(),
      now.getDate(),
      now.getHours() - index
    );

    return {
      id: `entry_${index}`,
      values: {
        hour_bucket: evenHour,
        event_name: events[index % events.length] ?? 'unknown event',
        count: Math.floor(Math.random() * 1000) + 1,
      },
      isLoadingRow: undefined,
    };
  });

  const hasNextPage = offset + pageSize < totalCount;
  return {
    columns: [
      { name: 'hour_bucket', type: 'date' as const },
      { name: 'event_name', type: 'string' as const },
      { name: 'count', type: 'number' as const },
    ],
    entries,
    pageInfo: {
      endCursor: hasNextPage ? `${offset + pageSize}` : null,
      hasNextPage,
      hasPreviousPage: offset > 0,
      startCursor: `${offset}`,
    },
    totalCount,
  };
}

export async function simulateQuery(
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _query: string,
  cursor: string | null
): Promise<InsightsFetchResult> {
  await new Promise((resolve) => setTimeout(resolve, 3000 + Math.random() * 1000));

  if (Math.random() <= 0.3) {
    throw new Error('Query timeout - please try a more specific query');
  }

  return getMockPage(cursor);
}
