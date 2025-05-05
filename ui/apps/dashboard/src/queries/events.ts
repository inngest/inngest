import { useQuery, type UseQueryResponse } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

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
        workflows {
          id
          slug
          name
          current {
            createdAt
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

type Event = {
  name: string;
  workflows: {
    id: string;
    slug: string;
    name: string;
    current: {
      createdAt: string;
    } | null;
  }[];
  usage: {
    total: number;
    data: {
      slot: string;
      count: number;
    }[];
  };
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

  const eventType = data?.events?.data[0];
  const dailyUsage: UsageItem[] | undefined = data?.events?.data[0]?.usage.data.map((d) => ({
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
