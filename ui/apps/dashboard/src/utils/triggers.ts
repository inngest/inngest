import type { Function } from '@inngest/components/types/function';

export function transformTriggers(
  rawTriggers: { eventName: string | null; schedule: string | null }[]
): Function['triggers'] {
  const triggers: Function['triggers'] = [];

  for (const trigger of rawTriggers) {
    if (trigger.eventName) {
      triggers.push({
        type: 'EVENT',
        value: trigger.eventName,
      });
    } else if (trigger.schedule) {
      triggers.push({
        type: 'CRON',
        value: trigger.schedule,
      });
    }
  }

  return triggers;
}
