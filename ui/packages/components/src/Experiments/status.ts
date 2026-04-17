export const ACTIVE_THRESHOLD_DAYS = 7;

export function isActive(lastSeen: Date): boolean {
  const threshold = new Date();
  threshold.setDate(threshold.getDate() - ACTIVE_THRESHOLD_DAYS);
  return lastSeen > threshold;
}
