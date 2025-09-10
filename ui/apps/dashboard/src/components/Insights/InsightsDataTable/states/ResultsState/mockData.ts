'use client';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';

// Mock dataset to exercise varied content shapes
// Includes: short text, long text, long text without spaces, numbers, dates,
// JSON strings, and nested JSON strings.

const lipsumShort = 'Lorem ipsum dolor sit amet, consectetur adipiscing elit.';

const lipsumLong =
  'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.';

// Intentionally no spaces to exercise word breaking in cells
const longNoSpaces =
  'LoremipsumdolorsitametconsecteturadipiscingelitSeddoeiusmodtemporincididuntutlaboreetdoloremagnaaliquaUt enimadminimveniam';

// Nested JSON object to stringify for JSON rendering scenarios
const nestedPayload = {
  rows: [
    {
      id: 1,
      customer: 'Alice',
      notes: {
        collapsed: 'Asked about integrations…',
        expanded: 'Asked about integrations with CRM. Wants to schedule a follow-up next week.',
        isExpanded: false,
      },
    },
  ],
};

const deepNested = {
  level1: {
    level2: {
      level3: {
        list: [
          { id: 1, label: 'Alpha', enabled: true },
          { id: 2, label: 'Beta', enabled: false },
        ],
        meta: { createdBy: 'system', tags: ['demo', 'nested', 'json'] },
      },
    },
  },
};

export const mockInsightsData: InsightsFetchResult = {
  columns: [
    { name: 'event_name', type: 'string' },
    { name: 'created_at', type: 'date' },
    { name: 'message', type: 'string' },
    { name: 'count', type: 'number' },
    { name: 'payload', type: 'string' }, // JSON string
    { name: 'notes', type: 'string' }, // JSON string
    { name: 'chars_no_spaces', type: 'string' },
    { name: 'long_text', type: 'string' },
  ],
  rows: [
    {
      id: '1',
      values: {
        event_name: 'test',
        created_at: new Date('2024-01-02T03:04:05.000Z').toISOString(),
        message: lipsumShort,
        count: 1,
        payload: JSON.stringify(nestedPayload),
        notes: JSON.stringify({ summary: 'Short note', priority: 'low' }),
        chars_no_spaces: longNoSpaces,
        long_text: lipsumLong,
      },
    },
    {
      id: '2',
      values: {
        event_name: 'example',
        created_at: new Date('2024-02-10T10:30:00.000Z').toISOString(),
        message: 'Nunc consequat interdum varius sit amet mattis vulputate enim nulla aliquet.',
        count: 12345,
        payload: JSON.stringify(deepNested),
        notes: JSON.stringify({
          collapsed: 'Inquired about API…',
          expanded: 'Inquired about API integration capabilities. Interested in a demo next week.',
          isExpanded: false,
        }),
        chars_no_spaces: longNoSpaces + longNoSpaces,
        long_text:
          lipsumLong +
          ' Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.',
      },
    },
    {
      id: '3',
      values: {
        event_name: 'metrics',
        created_at: new Date('2024-03-15T15:45:30.000Z').toISOString(),
        message: '',
        count: 0,
        payload: JSON.stringify({ totals: { a: 10, b: 20, c: 30 } }),
        notes: JSON.stringify({ info: 'Empty message row for edge case coverage' }),
        chars_no_spaces: 'AAAAABBBBBCCCCCDDDDDEEEEEFFFFF',
        long_text:
          'Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium.',
      },
    },
  ],
};

export default mockInsightsData;
