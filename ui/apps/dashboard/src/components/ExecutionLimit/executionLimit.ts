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
