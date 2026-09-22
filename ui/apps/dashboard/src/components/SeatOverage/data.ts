import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const seatOverageQuery = graphql(`
  query SeatOverageCheck {
    account {
      id
      entitlements: ents {
        userCount {
          limit
        }
        usage {
          userCount
        }
      }
    }
  }
`);

export type SeatOverageData = {
  hasExceeded: boolean;
  userCount: number;
  userLimit: number | null;
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

  const usage = entitlements.usage.userCount;
  const limit = userCount.limit;

  // null limit means no limit
  const hasExceeded = limit !== null && usage > limit;

  return {
    hasExceeded,
    userCount: usage,
    userLimit: limit,
  };
}
