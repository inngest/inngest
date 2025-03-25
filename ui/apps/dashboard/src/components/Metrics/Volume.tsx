import { useState } from 'react';
import { Error } from '@inngest/components/Error/Error';
import { RiArrowDownSFill, RiArrowRightSFill } from '@remixicon/react';

import { graphql } from '@/gql';
import { MetricsScope } from '@/gql/graphql';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { useEnvironment } from '../Environments/environment-context';
import { AUTO_REFRESH_INTERVAL } from './ActionMenu';
import { Backlog } from './Backlog';
import { AccountConcurrency } from './Concurrency';
import { type EntityLookup } from './Dashboard';
import { Feedback } from './Feedback';
import { RunsThrougput } from './RunsThroughput';
import { SdkThroughput } from './SdkThroughput';
import { StepsThroughput } from './StepsThroughput';

export type MetricsFilters = {
  from: Date;
  until?: Date;
  selectedApps?: string[];
  selectedFns?: string[];
  autoRefresh?: boolean;
  entities: EntityLookup;
  scope: MetricsScope;
  concurrencyLimit?: number;
};

const GetVolumeMetrics = graphql(`
  query VolumeMetrics(
    $workspaceId: ID!
    $from: Time!
    $functionIDs: [UUID!]
    $appIDs: [UUID!]
    $until: Time
    $scope: MetricsScope!
  ) {
    workspace(id: $workspaceId) {
      runsThroughput: scopedMetrics(
        filter: {
          name: "function_run_ended_total"
          scope: $scope
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
      sdkThroughputEnded: scopedMetrics(
        filter: {
          name: "sdk_req_ended_total"
          scope: $scope
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
      sdkThroughputStarted: scopedMetrics(
        filter: {
          name: "sdk_req_started_total"
          scope: $scope
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
      sdkThroughputScheduled: scopedMetrics(
        filter: {
          name: "sdk_req_scheduled_total"
          scope: $scope
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
      stepThroughput: scopedMetrics(
        filter: {
          name: "steps_running"
          scope: $scope
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
      backlog: scopedMetrics(
        filter: {
          name: "steps_scheduled"
          scope: $scope
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
      stepRunning: scopedMetrics(
        filter: {
          name: "steps_running"
          scope: $scope
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
      concurrency: scopedMetrics(
        filter: {
          name: "concurrency_limit_reached_total"
          scope: $scope
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
  }
`);

export const MetricsVolume = ({
  from,
  until,
  selectedApps = [],
  selectedFns = [],
  autoRefresh = false,
  entities,
  scope,
  concurrencyLimit,
}: MetricsFilters) => {
  const [volumeOpen, setVolumeOpen] = useState(true);

  const env = useEnvironment();

  const variables = {
    workspaceId: env.id,
    from: from.toISOString(),
    appIDs: selectedApps,
    functionIDs: selectedFns,
    until: until ? until.toISOString() : null,
    scope,
  };

  const { error, data } = useSkippableGraphQLQuery({
    skip: !env.id,
    query: GetVolumeMetrics,
    pollIntervalInMilliseconds: autoRefresh ? AUTO_REFRESH_INTERVAL * 1000 : 0,
    variables,
  });

  error && console.error('Error fetcthing metrics data for', variables, error);

  return (
    <div className="item-start flex h-full w-full flex-col items-start">
      <div
        className="text-subtle my-4 flex w-full cursor-pointer flex-row items-center justify-start gap-x-2 text-xs uppercase"
        onClick={() => setVolumeOpen(!volumeOpen)}
      >
        {volumeOpen ? <RiArrowDownSFill /> : <RiArrowRightSFill />}
        <div>Volume</div>

        <hr className="border-subtle w-full" />
      </div>
      {volumeOpen && (
        <>
          {error && <Error message="There was an error fetching volume metrics data." />}

          <div className="relative grid w-full auto-cols-max grid-cols-1 gap-2 overflow-hidden md:grid-cols-2">
            <RunsThrougput workspace={data?.workspace} entities={entities} />
            <StepsThroughput workspace={data?.workspace} entities={entities} />
            <div className="col-span-2 flex flex-row flex-wrap gap-2 overflow-hidden md:flex-nowrap">
              <SdkThroughput workspace={data?.workspace} />
              <Backlog workspace={data?.workspace} entities={entities} />
            </div>
            <div className="col-span-2 flex flex-row flex-wrap gap-2 overflow-hidden md:flex-nowrap">
              <AccountConcurrency
                workspace={data?.workspace}
                entities={entities}
                concurrencyLimit={concurrencyLimit}
              />
              <Feedback />
            </div>
            <div className="col-span-2 flex flex-row flex-wrap gap-2 overflow-hidden md:flex-nowrap"></div>
          </div>
        </>
      )}
    </div>
  );
};
