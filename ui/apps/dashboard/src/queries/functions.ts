import { Client, useQuery, type UseQueryResponse } from 'urql';

import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import { graphql } from '@/gql';
import type { GetFunctionQuery, WorkflowVersion } from '@/gql/graphql';
import { useEnvironment } from '@/queries/environments';

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
            version
            description
            validFrom
            validTo
            workflowType
            throttlePeriod
            throttleCount
            alerts {
              workflowID
            }
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
          workflowID
          version
          config
          retries
          validFrom
          validTo
          description
          updatedAt
          triggers {
            eventName
            schedule
            condition
            nextRun
          }
          deploy {
            id
            createdAt
          }
        }
        url
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
  environmentSlug: string;
  functionSlug: string;
};

export const useFunction = ({
  environmentSlug,
  functionSlug,
}: UseFunctionParams): UseQueryResponse<
  GetFunctionQuery,
  { environmentID: string; slug: string }
> => {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [result, refetch] = useQuery({
    query: GetFunctionDocument,
    variables: {
      environmentID: environment?.id!,
      slug: functionSlug,
    },
    pause: !environment?.id,
  });

  return [{ ...result, fetching: isFetchingEnvironment || result.fetching }, refetch];
};

export const FunctionVersionFragment = graphql(`
  fragment FunctionVersion on WorkflowVersion {
    version
    validFrom
    validTo
    triggers {
      eventName
      schedule
    }
    deploy {
      id
    }
  }
`);
const GetFunctionVersionsDocument = graphql(`
  query GetFunctionVersions($slug: String!, $environmentID: ID!) {
    workspace(id: $environmentID) {
      workflow: workflowBySlug(slug: $slug) {
        archivedAt
        current {
          ...FunctionVersion
        }
        previous {
          ...FunctionVersion
        }
      }
    }
  }
`);

type UseFunctionVersionsParams = {
  environmentSlug: string;
  functionSlug: string;
};

export const useFunctionVersions = ({
  environmentSlug,
  functionSlug,
}: UseFunctionVersionsParams): UseQueryResponse<WorkflowVersion[]> => {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [result, refetch] = useQuery({
    query: GetFunctionVersionsDocument,
    variables: {
      environmentID: environment?.id!,
      slug: functionSlug,
    },
    pause: !environment?.id,
  });

  const { data } = result;
  const versions: WorkflowVersion[] = data?.workspace.workflow?.current
    ? ([
        data?.workspace.workflow?.current,
        ...data?.workspace.workflow?.previous,
      ] as WorkflowVersion[])
    : data?.workspace.workflow?.previous
    ? (data?.workspace.workflow?.previous as WorkflowVersion[])
    : [];

  return [
    { ...result, data: versions, fetching: isFetchingEnvironment || result.fetching },
    refetch,
  ];
};

const GetFunctionUsageDocument = graphql(`
  query GetFunctionUsage($id: ID!, $environmentID: ID!, $startTime: Time!, $endTime: Time!) {
    workspace(id: $environmentID) {
      workflow(id: $id) {
        dailyStarts: usage(opts: { from: $startTime, to: $endTime }, event: "started") {
          period
          total
          asOf
          data {
            slot
            count
          }
        }
        dailyFailures: usage(opts: { from: $startTime, to: $endTime }, event: "errored") {
          period
          total
          asOf
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
  environmentSlug: string;
  functionSlug: string;
  timeRange: TimeRange;
};

export const useFunctionUsage = ({
  environmentSlug,
  functionSlug,
  timeRange,
}: UseFunctionUsageParams): UseQueryResponse<UsageItem[]> => {
  const [{ data: functionData }] = useFunction({ environmentSlug, functionSlug });
  const functionId = functionData?.workspace.workflow?.id;

  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });

  const [{ data, ...rest }, refetch] = useQuery({
    query: GetFunctionUsageDocument,
    variables: {
      environmentID: environment?.id!,
      id: functionId!,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
    pause: !functionId || !environment?.id,
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

  return [{ ...rest, data: usage, fetching: isFetchingEnvironment || rest.fetching }, refetch];
};
