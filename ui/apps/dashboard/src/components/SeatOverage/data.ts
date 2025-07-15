import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const seatOverageQuery = graphql(`
  query SeatOverageCheck {
    account {
      id
      hasExceededUserLimit
      entitlements {
        userCount {
          usage
          limit
        }
      }
    }
  }
`);

export type SeatOverageData = {
  hasExceeded: boolean;
  userCount: number;
  userLimit: number;
};

export function useSeatOverageCheck() {
  return useGraphQLQuery({
    query: seatOverageQuery,
    variables: {},
  });
}

export function parseSeatOverageData(data: any): SeatOverageData | null {
  if (!data?.account) return null;

  const { hasExceededUserLimit, entitlements } = data.account;
  const { userCount } = entitlements;

  return {
    hasExceeded: hasExceededUserLimit,
    userCount: userCount.usage,
    userLimit: userCount.limit || 0,
  };
}
