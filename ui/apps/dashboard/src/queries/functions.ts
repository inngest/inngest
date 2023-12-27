import { Client, useQuery, type UseQueryResponse } from 'urql';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { graphql } from '@/gql';
import type { GetFunctionQuery, WorkflowVersion } from '@/gql/graphql';

const GetFunctionsUsageDocument = graphql(`
  query GetFunctionsUsage($environmentID: ID!, $page: Int, $archived: Boolean) {
    workspace(id: $environmentID) {
      workflows(archived: $archived) @paginated(perPage: 50, page: $page) {
        page {
          page
          perPage
          totalItems
          totalPages
        }
        data {
          id
          slug
          dailyStarts: usage(opts: { period: "hour", range: "day" }, event: "started") {
            total
            data {
              count
            }
          }
          dailyFailures: usage(opts: { period: "hour", range: "day" }, event: "errored") {
            total
            data {
              count
            }
          }
        }
      }
    }
  }
`);

const GetFunctionsDocument = graphql(`
  query GetFunctions($environmentID: ID!, $page: Int, $archived: Boolean) {
    workspace(id: $environmentID) {
      workflows(archived: $archived) @paginated(perPage: 50, page: $page) {
        page {
          page
          perPage
          totalItems
          totalPages
        }
        data {
          appName
          id
          slug
          name
          isArchived
          current {
            triggers {
              eventName
              schedule
            }
          }
        }
      }
    }
  }
`);

export function getFunctions(args: {
  client: Client;
  environmentID: string;
  isArchived: boolean;
  page: number;
}) {
  return args.client
    .query(GetFunctionsDocument, {
      environmentID: args.environmentID,
      archived: args.isArchived,
      page: args.page,
    })
    .toPromise();
}

const GetFunctionDocument = graphql(`
  query GetFunction($slug: String!, $environmentID: ID!) {
    workspace(id: $environmentID) {
      id
      workflow: workflowBySlug(slug: $slug) {
        id
        name
        slug
        isArchived
        appName
        current {
          triggers {
            eventName
            schedule
            condition
          }
          deploy {
            id
            createdAt
          }
        }
        failureHandler {
          slug
          name
        }
        configuration {
          cancellations {
            event
            timeout
            condition
          }
          retries {
            value
            isDefault
          }
          priority
          eventsBatch {
            maxSize
            timeout
          }
          concurrency {
            scope
            limit {
              value
              isPlanLimit
            }
            key
          }
          rateLimit {
            limit
            period
            key
          }
          debounce {
            period
            key
          }
        }
      }
    }
  }
`);

type UseFunctionParams = {
  functionSlug: string;
};

export const useFunction = ({
  functionSlug,
}: UseFunctionParams): UseQueryResponse<
  GetFunctionQuery,
  { environmentID: string; slug: string }
> => {
  const environment = useEnvironment();
  const [result, refetch] = useQuery({
    query: GetFunctionDocument,
    variables: {
      environmentID: environment.id,
      slug: functionSlug,
    },
  });

  return [{ ...result, fetching: result.fetching }, refetch];
};

const GetFunctionUsageDocument = graphql(`
  query GetFunctionUsage($id: ID!, $environmentID: ID!, $startTime: Time!, $endTime: Time!) {
    workspace(id: $environmentID) {
      workflow(id: $id) {
        dailyStarts: usage(opts: { from: $startTime, to: $endTime }, event: "started") {
          period
          total
          data {
            slot
            count
          }
        }
        dailyFailures: usage(opts: { from: $startTime, to: $endTime }, event: "errored") {
          period
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

export function getFunctionUsages(args: {
  client: Client;
  environmentID: string;
  isArchived: boolean;
  page: number;
}) {
  return args.client
    .query(GetFunctionsUsageDocument, {
      environmentID: args.environmentID,
      archived: args.isArchived,
      page: args.page,
    })
    .toPromise();
}

type UsageItem = {
  name: string;
  values: {
    totalRuns: number;
    successes: number;
    failures: number;
  };
};

type UseFunctionUsageParams = {
  functionSlug: string;
  timeRange: TimeRange;
};

export const useFunctionUsage = ({
  functionSlug,
  timeRange,
}: UseFunctionUsageParams): UseQueryResponse<UsageItem[]> => {
  const environment = useEnvironment();
  const [{ data: functionData }] = useFunction({ functionSlug });
  const functionId = functionData?.workspace.workflow?.id;

  const [{ data, ...rest }, refetch] = useQuery({
    query: GetFunctionUsageDocument,
    variables: {
      environmentID: environment.id,
      id: functionId!,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
    pause: !functionId,
  });

  // Combine usage arrays into single array
  let usage: UsageItem[] = [];
  const starts = data?.workspace.workflow?.dailyStarts;
  const failures = data?.workspace.workflow?.dailyFailures;
  if (starts && failures) {
    usage = starts.data.map((d, idx) => {
      const failureCount = failures.data[idx]?.count || 0;
      return {
        name: d.slot,
        values: {
          totalRuns: d.count,
          successes: d.count - failureCount,
          failures: failureCount,
        },
      };
    });
  }

  return [{ ...rest, data: usage, fetching: rest.fetching }, refetch];
};
