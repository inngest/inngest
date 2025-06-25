import cronstrue from 'cronstrue';

/**
 * Converts a cron expression to a human-readable description.
 * Falls back to the original cron expression if parsing fails.
 */
export function getHumanReadableCron(cronExpression: string): string {
  try {
    return cronstrue.toString(cronExpression);
  } catch {
    return cronExpression;
  }
}
