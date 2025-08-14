import useGetUsageChartData from '@/components/Billing/Usage/useGetUsageChartData';
import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const executionOverageQuery = graphql(`
  query ExecutionOverageCheck {
    account {
      id
      entitlements {
        executions {
          limit
        }
      }
    }
  }
`);

export type ExecutionOverageData = {
  hasExceeded: boolean;
  executionCount: number;
  executionLimit: number | null;
};

export function useExecutionOverageCheck() {
  const { data: rawData, error } = useGraphQLQuery({
    query: executionOverageQuery,
    variables: {},
  });

  // Get current month's execution usage data
  const { data: usageData } = useGetUsageChartData({
    selectedPeriod: 'current',
    type: 'execution',
  });

  return {
    data: rawData,
    usageData,
    error,
  };
}

export function parseExecutionOverageData(
  data: any,
  usageData: any[]
): ExecutionOverageData | null {
  if (!data?.account) return null;

  const { entitlements } = data.account;
  const { executions } = entitlements;

  const limit = executions.limit;

  // Calculate current usage by summing the time series data
  const usage = usageData.reduce((sum, point) => {
    return sum + (point.value || 0);
  }, 0);

  // null limit means no limit
  const hasExceeded = limit !== null && usage > limit;

  return {
    hasExceeded,
    executionCount: usage,
    executionLimit: limit,
  };
}
