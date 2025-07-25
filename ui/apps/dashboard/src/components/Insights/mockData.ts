const EVENT_NAMES = [
  'action.created',
  'action.validated',
  'auth.login',
  'brand/manual-add-off-platform-brands-created',
  'clerk/organization/MembershipDeleted',
  'data.synced',
  'notification.sent',
  'payment.processed',
  'user.registered',
  'workflow.completed',
] as const;

const FUNCTION_NAMES = [
  'analyze-metrics',
  'cleanup-temp-files',
  'create-webhook',
  'generate-report',
  'process-payment',
  'schedule-backup',
  'send-notification',
  'sync-data',
  'update-profile',
  'validate-user',
] as const;

export const FIELD_NAMES = {
  EVENT_NAME: 'event_name',
  FUNCTION_TRIGGERED: 'function_triggered',
  RECEIVED_AT: 'received_at',
} as const;

export const FIELD_DISPLAY_NAMES = {
  [FIELD_NAMES.EVENT_NAME]: 'Event Name',
  [FIELD_NAMES.FUNCTION_TRIGGERED]: 'Function Triggered',
  [FIELD_NAMES.RECEIVED_AT]: 'Received At',
} as const;

export interface InsightData {
  [FIELD_NAMES.EVENT_NAME]: (typeof EVENT_NAMES)[number];
  [FIELD_NAMES.FUNCTION_TRIGGERED]: (typeof FUNCTION_NAMES)[number];
  [FIELD_NAMES.RECEIVED_AT]: string;
}

function getRandomElement<T>(array: readonly T[]): T {
  const index = Math.floor(Math.random() * array.length);
  return array[index] as T;
}

function getRandomRecentDate(): string {
  const daysAgo = Math.random() * 7;
  const millisecondsAgo = daysAgo * 24 * 60 * 60 * 1000;
  return new Date(Date.now() - millisecondsAgo).toLocaleString();
}

export function generateInsightsMockData(n: number): InsightData[] {
  const data: InsightData[] = [];

  for (let i = 0; i < n; i++) {
    data.push({
      [FIELD_NAMES.EVENT_NAME]: getRandomElement(EVENT_NAMES),
      [FIELD_NAMES.FUNCTION_TRIGGERED]: getRandomElement(FUNCTION_NAMES),
      [FIELD_NAMES.RECEIVED_AT]: getRandomRecentDate(),
    });
  }

  return data;
}
