import type { ExecutionLimitCheckQuery } from '@/gql/graphql';

export type ExecutionCap = {
  usage: number;
  limit: number;
  enforced: boolean;
  overageAllowed: boolean;
};

// Display policy is separate from whether execution is actually blocked.
export function shouldShowExecutionLimit(cap: ExecutionCap): boolean {
  return !cap.overageAllowed;
}

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

export type UsageKind = 'default' | 'caution' | 'warning' | 'error';

export function usageKind(band: UsageBand): UsageKind {
  if (band === 'capped') return 'error';
  if (band === 'under50') return 'default';
  if (band === '50') return 'caution';
  if (band === '75') return 'warning';
  return 'error';
}

export function pillContent(
  band: UsageBand,
  isVercel: boolean,
): { kind: 'caution' | 'warning' | 'error'; text: string } | null {
  const kind = usageKind(band);
  if (kind === 'default') return null;

  if (band === 'capped') {
    return {
      kind,
      text: isVercel
        ? 'Hobby account executions limit reached. Change configurations on the Vercel Integrations Settings page to upgrade and resume using Inngest.'
        : 'Hobby account executions limit reached. Upgrade your plan to continue using Inngest.',
    };
  }

  return {
    kind,
    text: isVercel
      ? 'Hobby account executions limit almost reached. Change configurations on the Vercel Integrations Settings page to upgrade and avoid disruption.'
      : 'Hobby account executions limit almost reached. Upgrade your plan to continue using Inngest.',
  };
}

export type DismissalSurface = 'card';

const DISMISSAL_STORAGE_KEY_PREFIX: Record<DismissalSurface, string> = {
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
