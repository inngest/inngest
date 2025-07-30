import { type InsightsResult } from './types';

const COLUMNS = [
  { name: 'hour_bucket', type: 'date' as const },
  { name: 'event_name', type: 'string' as const },
  { name: 'count', type: 'number' as const },
];

const EVENT_NAMES = [
  'action-created',
  'user/profile-updated',
  'payment/subscription-renewed',
  'payment/subscription-cancelled',
  'user/login',
  'user/logout',
  'order/created',
  'order/completed',
  'notification/sent',
];

function generateEntry(id: number, hourOffset: number): any {
  const baseTime = new Date('2025-04-22T13:30:00.000Z');
  baseTime.setHours(baseTime.getHours() + hourOffset);

  return {
    id: id.toString(),
    values: {
      hour_bucket: baseTime.toISOString(),
      event_name: EVENT_NAMES[id % EVENT_NAMES.length],
      count: Math.floor(Math.random() * 200) + 10,
    },
  };
}

const TOTAL_COUNT = 150;

const PAGE_1: InsightsResult = {
  columns: COLUMNS,
  entries: Array.from({ length: 50 }, (_, i) => generateEntry(i + 1, Math.floor(i / 5))),
  totalCount: TOTAL_COUNT,
  pageInfo: {
    endCursor: 'cursor_50',
    hasNextPage: true,
    hasPreviousPage: false,
    startCursor: 'cursor_1',
  },
};

const PAGE_2: InsightsResult = {
  columns: COLUMNS,
  entries: Array.from({ length: 50 }, (_, i) => generateEntry(i + 51, Math.floor((i + 50) / 5))),
  totalCount: TOTAL_COUNT,
  pageInfo: {
    endCursor: 'cursor_100',
    hasNextPage: true,
    hasPreviousPage: true,
    startCursor: 'cursor_51',
  },
};

const PAGE_3: InsightsResult = {
  columns: COLUMNS,
  entries: Array.from({ length: 50 }, (_, i) => generateEntry(i + 101, Math.floor((i + 100) / 5))),
  totalCount: TOTAL_COUNT,
  pageInfo: {
    endCursor: 'cursor_150',
    hasNextPage: false,
    hasPreviousPage: true,
    startCursor: 'cursor_101',
  },
};

const EMPTY_PAGE: InsightsResult = {
  columns: [],
  entries: [],
  totalCount: 0,
  pageInfo: {
    endCursor: null,
    hasNextPage: false,
    hasPreviousPage: false,
    startCursor: null,
  },
};

export function getMockPage(cursor: string | null): InsightsResult {
  if (cursor === null) return PAGE_1;

  switch (cursor) {
    case 'cursor_50':
      return PAGE_2;
    case 'cursor_100':
      return PAGE_3;
    default:
      return EMPTY_PAGE;
  }
}
