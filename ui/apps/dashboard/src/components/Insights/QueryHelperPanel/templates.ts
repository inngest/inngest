import type { SavedQuery } from './types';

export const TEMPLATE_QUERIES: SavedQuery[] = [
  {
    id: 'event-types-per-hour',
    name: 'Event types per hour',
    text: 'SELECT\n  HOUR(ts) as hour,\n  name,\n  COUNT(*) as event_count\nFROM\n  events\nWHERE\n  event_ts > 1754424142590\nGROUP BY\n  hour, name\nORDER BY\n  hour, name DESC;',
    updatedOn: '1970-01-01T00:00:00.000Z',
  },
  {
    id: 'function-runs-by-hour-status',
    name: 'Function runs by hour and status',
    text: "SELECT\n  HOUR(queued_at) as start_hour,\n  status,\n  left(output.message, 50) as message,\n  COUNT(*)\nFROM\n  runs\nWHERE\n  app_name = 'app_name'\n  AND function_name = 'function_name'\n  AND queued_at > NOW() - INTERVAL 3 DAY\nGROUP BY\n  start_hour, status, message\nORDER BY\n  start_hour, status DESC;",
    updatedOn: '1970-01-01T00:00:00.000Z',
  },
];
