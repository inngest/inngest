import { pathCreator } from '@/utils/urls';

export const BURST_MULTIPLIER = 3;

export type ConcurrencyAlertState = 'boosted' | 'limited';

export type ConcurrencyAlertSurface = 'pill' | 'card' | 'card-collapsed';

export function alertState({
  boosted,
  pressured,
}: {
  boosted: boolean;
  pressured: boolean;
}): ConcurrencyAlertState | null {
  if (boosted) {
    return 'boosted';
  }
  return pressured ? 'limited' : null;
}

export function effectiveLimit({
  state,
  burstLimit,
  baseLimit,
}: {
  state: ConcurrencyAlertState;
  burstLimit: number | undefined;
  baseLimit: number | undefined;
}): number | undefined {
  return state === 'boosted' && burstLimit != null ? burstLimit : baseLimit;
}

export function minutesLeftLabel(minutes: number): string {
  return `${minutes} ${minutes === 1 ? 'min' : 'mins'}`;
}

export function usedRatio({
  budgetMinutes,
  minutesRemaining,
}: {
  budgetMinutes: number;
  minutesRemaining: number;
}): number {
  if (budgetMinutes <= 0) {
    return 1;
  }
  const ratio = (budgetMinutes - minutesRemaining) / budgetMinutes;
  return Math.min(Math.max(ratio, 0), 1);
}

export function concurrencyLinkProps({
  state,
  surface,
  isMarketplace,
  isFreePlan,
  marketplaceBillingURL,
}: {
  state: ConcurrencyAlertState;
  surface: ConcurrencyAlertSurface;
  isMarketplace: boolean;
  isFreePlan: boolean | undefined;
  marketplaceBillingURL: string | null | undefined;
}): { to: string } {
  if (state === 'boosted') {
    return {
      to: pathCreator.billingUsage({ ref: `app-concurrency-boost-${surface}` }),
    };
  }
  if (isMarketplace && marketplaceBillingURL) {
    return { to: marketplaceBillingURL };
  }
  const ref = `app-concurrency-limited-${surface}`;
  return {
    to: isFreePlan
      ? pathCreator.billing({ tab: 'plans', ref })
      : pathCreator.billing({ highlight: 'concurrency', ref }),
  };
}
