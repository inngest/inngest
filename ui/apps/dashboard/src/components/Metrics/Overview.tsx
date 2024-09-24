import { useState } from 'react';
import { Error } from '@inngest/components/Error/Error';
import { RiArrowDownSFill, RiArrowRightSFill } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { MetricsScope } from '@/gql/graphql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { AUTO_REFRESH_INTERVAL } from './ActionMenu';
import type { MetricsFilters } from './Dashboard';
import { FailedFunctions } from './FailedFunctions';
import { FunctionStatus } from './FunctionStatus';

const GetFunctionStatusMetrics = graphql(`
  query FunctionStatusMetrics(
    $workspaceId: ID!
    $from: Time!
    $functionIDs: [UUID!]
    $appIDs: [UUID!]
    $until: Time
    $scope: MetricsScope!
  ) {
    workspace(id: $workspaceId) {
      scheduled: scopedMetrics(
        filter: {
          name: "function_run_scheduled_total"
          scope: $scope
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          data {
            value
            bucket
          }
        }
      }
    }
    workspace(id: $workspaceId) {
      started: scopedMetrics(
        filter: {
          name: "function_run_started_total"
          scope: $scope
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          data {
            value
            bucket
          }
        }
      }
    }
    workspace(id: $workspaceId) {
      completed: scopedMetrics(
        filter: {
          name: "function_run_ended_total"
          scope: $scope
          groupBy: "status"
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          tagName
          tagValue
          data {
            value
            bucket
          }
        }
      }
    }
    workspace(id: $workspaceId) {
      completedByFunction: scopedMetrics(
        filter: {
          name: "function_run_ended_total"
          scope: FN
          groupBy: "status"
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          tagName
          tagValue
          data {
            value
            bucket
          }
        }
      }
    }
    workspace(id: $workspaceId) {
      totals: scopedFunctionStatus(
        filter: {
          name: "function_run_scheduled_total"
          scope: FN
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        queued
        running
        completed
        failed
        cancelled
        cancelled
        skipped
      }
    }
  }
`);

export const MetricsOverview = ({
  from,
  until,
  selectedApps = [],
  selectedFns = [],
  autoRefresh = false,
  entities, // dynamic based on scope
  functions,
}: MetricsFilters) => {
  const [overviewOpen, setOverviewOpen] = useState(true);
  const env = useEnvironment();

  const variables = {
    workspaceId: env.id,
    from: from.toISOString(),
    appIDs: selectedApps,
    functionIDs: selectedFns,
    until: until ? until.toISOString() : null,
    scope: !selectedApps.length && !selectedFns.length ? MetricsScope.App : MetricsScope.Fn,
  };

  const { data, error } = useGraphQLQuery({
    query: GetFunctionStatusMetrics,
    pollIntervalInMilliseconds: autoRefresh ? AUTO_REFRESH_INTERVAL * 1000 : 0,
    variables,
  });

  error && console.error('Error fetcthing metrics data for', variables, error);

  return (
    <div className="item-start flex h-full w-full flex-col items-start">
      <div className="text-subtle my-4 flex w-full flex-row items-center justify-start gap-x-2 text-xs uppercase">
        {overviewOpen ? (
          <RiArrowDownSFill className="cursor-pointer" onClick={() => setOverviewOpen(false)} />
        ) : (
          <RiArrowRightSFill className="cursor-pointer" onClick={() => setOverviewOpen(true)} />
        )}
        <div>Overview</div>

        <hr className="border-subtle w-full" />
      </div>
      {overviewOpen && (
        <>
          {error && <Error message="There was an error fetching overview metrics data." />}
          <div className="relative flex w-full flex-row flex-wrap items-center justify-start gap-2 overflow-hidden md:flex-nowrap">
            <FunctionStatus totals={data?.workspace.totals} />
            <FailedFunctions
              workspace={data?.workspace}
              entities={entities}
              functions={functions}
            />
          </div>
        </>
      )}
    </div>
  );
};
