import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const seatOverageQuery = graphql(`
  query SeatOverageCheck {
    account {
      id
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

  const { entitlements } = data.account;
  const { userCount } = entitlements;

  const usage = userCount.usage;
  const limit = userCount.limit || 0;

  // limit of -1 means no limit
  const hasExceeded = limit >= 0 && usage > limit;

  return {
    hasExceeded,
    userCount: usage,
    userLimit: limit,
  };
}
