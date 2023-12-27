import { useQuery, type UseQueryResponse } from 'urql';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { graphql } from '@/gql';
import type { Event, GetEventTypesQuery } from '@/gql/graphql';

const GetEventTypesDocument = graphql(`
  query GetEventTypes($environmentID: ID!, $page: Int) {
    workspace(id: $environmentID) {
      events @paginated(perPage: 50, page: $page) {
        data {
          name
          functions: workflows {
            id
            slug
            name
          }
          dailyVolume: usage(opts: { period: "hour", range: "day" }) {
            total
            data {
              count
            }
          }
        }
        page {
          page
          totalPages
        }
      }
    }
  }
`);

type UseEventTypesParams = {
  page?: number;
};

export const useEventTypes = ({
  page = 0,
}: UseEventTypesParams): UseQueryResponse<GetEventTypesQuery, { page?: number }> => {
  const env = useEnvironment();
  const [result, refetch] = useQuery({
    query: GetEventTypesDocument,
    variables: {
      environmentID: env.id,
      page,
    },
  });
  return [{ ...result, fetching: result.fetching }, refetch];
};

const GetEventTypeDocument = graphql(`
  query GetEventType($eventName: String, $environmentID: ID!) {
    events(query: { name: $eventName, workspaceID: $environmentID }) {
      data {
        name
        usage(opts: { period: "hour", range: "day" }) {
          total
          data {
            slot
            count
          }
        }
      }
    }
  }
`);

// Use a standard shape for the bar chart
type UsageItem = {
  name: string;
  values: {
    count: number;
  };
};

type UseEventTypeParams = {
  name: string;
};

export const useEventType = ({
  name,
}: UseEventTypeParams): UseQueryResponse<{
  eventType: Event | undefined;
  dailyUsage: UsageItem[] | undefined;
}> => {
  const environment = useEnvironment();
  const [{ data, ...rest }, refetch] = useQuery({
    query: GetEventTypeDocument,
    variables: {
      environmentID: environment.id,
      eventName: name,
    },
  });

  const eventType = data?.events?.data?.[0] as Event | undefined;
  const dailyUsage: UsageItem[] | undefined = data?.events?.data?.[0]?.usage.data.map((d) => ({
    name: d.slot,
    values: {
      count: d.count,
    },
  }));

  return [
    {
      data: {
        eventType,
        dailyUsage,
      },
      ...rest,
      fetching: rest.fetching,
    },
    refetch,
  ];
};
