import { useState } from 'react';
import { RiArrowDownSFill, RiArrowRightSFill } from '@remixicon/react';

import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { useEnvironment } from '../Environments/environment-context';
import { AUTO_REFRESH_INTERVAL } from './ActionMenu';
import type { FunctionLookup } from './Dashboard';
import { RunsThrougput } from './RunsThroughput';
import { StepsThroughput } from './StepsThroughput';

export type MetricsFilters = {
  from: Date;
  until?: Date;
  selectedApps?: string[];
  selectedFns?: string[];
  autoRefresh?: boolean;
  functions: FunctionLookup;
};

const GetVolumeMetrics = graphql(`
  query VolumeMetrics(
    $workspaceId: ID!
    $from: Time!
    $functionIDs: [UUID!]
    $appIDs: [UUID!]
    $until: Time
  ) {
    workspace(id: $workspaceId) {
      runsThroughput: scopedMetrics(
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
      sdkThroughput: scopedMetrics(
        filter: {
          name: "sdk_req_ended_total"
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
      stepThroughput: scopedMetrics(
        filter: {
          name: "step_output_bytes_total"
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
  }
`);

export const MetricsVolume = ({
  from,
  until,
  selectedApps = [],
  selectedFns = [],
  autoRefresh = false,
  functions,
}: MetricsFilters) => {
  const [volumeOpen, setVolumeOpen] = useState(true);

  const env = useEnvironment();

  const variables = {
    workspaceId: env.id,
    from: from.toISOString(),
    appIDs: selectedApps,
    functionIDs: selectedFns,
    until: until ? until.toISOString() : null,
  };

  const { data } = useGraphQLQuery({
    query: GetVolumeMetrics,
    pollIntervalInMilliseconds: autoRefresh ? AUTO_REFRESH_INTERVAL * 1000 : 0,
    variables,
  });

  return (
    <div className="bg-canvasSubtle item-start flex h-full w-full flex-col items-start">
      <div className="text-subtle my-4 flex w-full flex-row items-center justify-start gap-x-2 text-xs uppercase">
        {volumeOpen ? (
          <RiArrowDownSFill className="cursor-pointer" onClick={() => setVolumeOpen(false)} />
        ) : (
          <RiArrowRightSFill className="cursor-pointer" onClick={() => setVolumeOpen(true)} />
        )}
        <div>Volume</div>

        <hr className="border-subtle w-full" />
      </div>
      {volumeOpen && (
        <div className="relative flex w-full flex-row items-center justify-start gap-2 overflow-hidden">
          <RunsThrougput workspace={data?.workspace} functions={functions} />
          <StepsThroughput workspace={data?.workspace} functions={functions} />
        </div>
      )}
    </div>
  );
};
