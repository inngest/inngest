import type { ExecutionLimitCheckQuery } from '@/gql/graphql';

type LegacyEntitlements = ExecutionLimitCheckQuery['account']['entitlements'];

// The legacy query reads usage separately from the executions entitlement.
export type LegacyExecutions = LegacyEntitlements['executions'] & {
  usage: LegacyEntitlements['usage']['executions'];
};

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
}: LegacyExecutions): ExecutionCap | null {
  if (limit === null) return null;
  return { usage, limit, overageAllowed, enforced: true };
}
