import type { ExecutionLimitCheckQuery } from '@/gql/graphql';

export type ExecutionCap = {
  usage: number;
  limit: number;
  enforced: boolean;
  overageAllowed: boolean;
};

export function isExecutionCapped({
  usage,
  limit,
  enforced,
  overageAllowed,
}: ExecutionCap): boolean {
  return enforced && !overageAllowed && usage >= limit;
}

export function legacyExecutionCap({
  usage,
  limit,
  overageAllowed,
}: ExecutionLimitCheckQuery['account']['entitlements']['executions']): ExecutionCap | null {
  if (limit === null) return null;
  return { usage, limit, overageAllowed, enforced: true };
}

export type UsageBand = 'under50' | '50' | '75' | '90' | 'capped';

export function usageBand({
  usage,
  limit,
  isCapped,
}: {
  usage: number;
  limit: number;
  isCapped: boolean;
}): UsageBand {
  if (isCapped) return 'capped';
  if (limit <= 0) return 'under50';

  const ratio = usage / limit;
  if (ratio >= 0.9) return '90';
  if (ratio >= 0.75) return '75';
  if (ratio >= 0.5) return '50';
  return 'under50';
}

export type DismissalSurface = 'banner' | 'card';

const DISMISSAL_STORAGE_KEY_PREFIX: Record<DismissalSurface, string> = {
  banner: 'dismissedExecutionLimitBanner',
  card: 'dismissedExecutionLimitCard',
};

export function dismissalStorageKey(
  surface: DismissalSurface,
  accountID: string,
): string {
  return `${DISMISSAL_STORAGE_KEY_PREFIX[surface]}:${accountID}`;
}

export const DISMISSAL_LIFETIME_MS = 24 * 60 * 60 * 1000;

export function isDismissalActive(
  dismissedAt: number | null,
  now: number,
): boolean {
  if (dismissedAt === null || !Number.isFinite(dismissedAt)) return false;
  if (dismissedAt > now) return false;

  return now - dismissedAt < DISMISSAL_LIFETIME_MS;
}

export function readDismissedAt(
  surface: DismissalSurface,
  accountID: string,
): number | null {
  const key = dismissalStorageKey(surface, accountID);

  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return null;

    const parsed = Number(raw);
    return Number.isFinite(parsed) ? parsed : null;
  } catch (error) {
    console.warn(`error reading localStorage key "${key}":`, error);
    return null;
  }
}

export function writeDismissedAt(
  surface: DismissalSurface,
  accountID: string,
  now: number,
): void {
  const key = dismissalStorageKey(surface, accountID);

  try {
    window.localStorage.setItem(key, String(now));
  } catch (error) {
    console.warn(`error writing localStorage key "${key}":`, error);
  }
}
