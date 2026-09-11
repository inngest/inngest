import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const executionLimitQuery = graphql(`
  query ExecutionLimitCheck {
    account {
      id
      entitlements {
        executions {
          usage
          limit
          overageAllowed
        }
      }
    }
  }
`);

type ExecutionLimitData = {
  isCapped: boolean;
  usedExecutions: number;
  executionLimit: number;
};

export function useExecutionLimit(): ExecutionLimitData | null {
  const res = useGraphQLQuery({ query: executionLimitQuery, variables: {} });
  if (!res.data) return null;

  const { usage, limit, overageAllowed } =
    res.data.account.entitlements.executions;
  if (limit === null) return null;

  return {
    isCapped: !overageAllowed && usage >= limit,
    usedExecutions: usage,
    executionLimit: limit,
  };
}
