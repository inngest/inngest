export const ACTIVE_THRESHOLD_DAYS = 7;

/**
 * Cutoff before which `lastSeen` counts as inactive. Call once per render
 * and reuse — the result is a `Date` allocation plus `setDate` each call.
 */
export function getActiveThreshold(): Date {
  const threshold = new Date();
  threshold.setDate(threshold.getDate() - ACTIVE_THRESHOLD_DAYS);
  return threshold;
}

export function isActive(lastSeen: Date, threshold?: Date): boolean {
  return lastSeen > (threshold ?? getActiveThreshold());
}
