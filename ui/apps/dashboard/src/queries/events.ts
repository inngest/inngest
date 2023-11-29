import { useQuery, type UseQueryResponse } from 'urql';

import { graphql } from '@/gql';
import type { Event, GetEventTypesQuery } from '@/gql/graphql';
import { useEnvironment } from '@/queries/environments';

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
  environmentSlug: string;
  page?: number;
};

export const useEventTypes = ({
  environmentSlug,
  page = 0,
}: UseEventTypesParams): UseQueryResponse<
  GetEventTypesQuery,
  { environmentID: string; page?: number }
> => {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });

  const [result, refetch] = useQuery({
    query: GetEventTypesDocument,
    variables: {
      environmentID: environment?.id!,
      page,
    },
    pause: !environment?.id,
  });
  return [{ ...result, fetching: isFetchingEnvironment || result.fetching }, refetch];
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
  environmentSlug: string;
  name: string;
};

export const useEventType = ({
  environmentSlug,
  name,
}: UseEventTypeParams): UseQueryResponse<{
  eventType: Event | undefined;
  dailyUsage: UsageItem[] | undefined;
}> => {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [{ data, ...rest }, refetch] = useQuery({
    query: GetEventTypeDocument,
    variables: {
      environmentID: environment?.id!,
      eventName: name,
    },
    pause: !environment?.id,
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
      fetching: isFetchingEnvironment || rest.fetching,
    },
    refetch,
  ];
};
