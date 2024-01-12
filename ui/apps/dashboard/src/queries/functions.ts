import { Client, useQuery, type UseQueryResponse } from 'urql';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { graphql } from '@/gql';
import type { GetFunctionQuery, WorkflowVersion } from '@/gql/graphql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const GetFunctionsUsageDocument = graphql(`
  query GetFunctionsUsage($environmentID: ID!, $page: Int, $archived: Boolean, $pageSize: Int) {
    workspace(id: $environmentID) {
      workflows(archived: $archived) @paginated(perPage: $pageSize, page: $page) {
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
  query GetFunctions($environmentID: ID!, $page: Int, $archived: Boolean, $pageSize: Int) {
    workspace(id: $environmentID) {
      workflows(archived: $archived) @paginated(perPage: $pageSize, page: $page) {
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

export function useFunctionsPage({
  archived,
  envID,
  page,
}: {
  archived: boolean;
  envID: string;
  page: number;
}) {
  const pageSize = 5;
  const res = useGraphQLQuery({
    query: GetFunctionsDocument,
    variables: {
      archived,
      environmentID: envID,
      page,
      pageSize,
    },
  });
  if (!res.data) {
    return {
      ...res,
      data: undefined,
    };
  }

  return {
    ...res,
    data: {
      functions: res.data.workspace.workflows.data.map((fn) => {
        let triggers: { type: 'event' | 'schedule'; value: string }[] = [];
        if (fn.current) {
          for (const trigger of fn.current.triggers) {
            if (trigger.schedule) {
              triggers.push({
                type: 'schedule',
                value: trigger.schedule,
              });
            } else if (trigger.eventName) {
              triggers.push({
                type: 'event',
                value: trigger.eventName,
              });
            }
          }
        }

        return {
          ...fn,
          failureRate: undefined,
          isActive: !fn.isArchived,
          triggers,
          usage: undefined,
        };
      }),
      page: {
        ...res.data.workspace.workflows.page,
        hasNextPage: res.data.workspace.workflows.data.length === pageSize,
      },
    },
  };
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

export async function getFunctionUsagesPage(args: {
  archived: boolean;
  client: Client;
  envID: string;
  page: number;
}) {
  const pageSize = 2;

  const res = await args.client
    .query(GetFunctionsUsageDocument, {
      environmentID: args.envID,
      archived: args.archived,
      page: args.page,
      pageSize,
    })
    .toPromise();
  if (res.error) {
    throw res.error;
  }
  if (!res.data) {
    throw new Error('no data returned');
  }

  res.data.workspace;

  return {
    ...res,
    data: {
      functions: res.data.workspace.workflows.data.map((fn) => {
        const dailyStartCount = fn.dailyStarts.total;
        const dailyFailureCount = fn.dailyFailures.total;

        // Calculates the daily failure rate percentage and rounds it up to 2 decimal places
        const failureRate =
          dailyStartCount === 0
            ? 0
            : Math.round((dailyFailureCount / dailyStartCount) * 10000) / 100;

        // Creates an array of objects containing the start and failure count for each usage slot (1 hour)
        const slots = fn.dailyStarts.data.map((usageSlot, index) => ({
          startCount: usageSlot.count,
          failureCount: fn.dailyFailures.data[index]?.count ?? 0,
        }));

        const usage = {
          slots,
          total: dailyStartCount,
        };

        return {
          failureRate,
          slug: fn.slug,
          usage,
        };
      }),
      page: res.data.workspace.workflows.page,
    },
  };
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
