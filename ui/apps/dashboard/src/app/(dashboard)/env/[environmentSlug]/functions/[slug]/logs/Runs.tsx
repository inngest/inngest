'use client';

import React, { useState } from 'react';
import { useOrganization, useUser } from '@clerk/nextjs';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import NewReplayButton from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/NewReplayButton';
import { ClientFeatureFlag } from '@/components/FeatureFlags/ClientFeatureFlag';
import { graphql } from '@/gql';
import { FunctionRunStatus, FunctionRunTimeField } from '@/gql/graphql';
import FunctionRunList from './FunctionRunList';
import StatusFilter from './StatusFilter';
import TimeRangeFilter, {
  defaultTimeField,
  defaultTimeRange,
  type TimeRange,
} from './TimeRangeFilter';

type RunsPageProps = {
  params: {
    slug: string;
  };
};

const GetFunctionRunsCountDocument = graphql(`
  query GetFunctionRunsCount(
    $environmentID: ID!
    $functionSlug: String!
    $functionRunStatuses: [FunctionRunStatus!]
    $timeRangeStart: Time!
    $timeRangeEnd: Time!
    $timeField: FunctionRunTimeField!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        runs: runsV2(
          filter: {
            status: $functionRunStatuses
            lowerTime: $timeRangeStart
            upperTime: $timeRangeEnd
            timeField: $timeField
          }
        ) {
          totalCount
        }
      }
    }
  }
`);

export default function RunsPage({ params }: RunsPageProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [selectedStatuses, setSelectedStatuses] = useState<FunctionRunStatus[]>([]);
  const [selectedTimeField, setSelectedTimeField] =
    useState<FunctionRunTimeField>(defaultTimeField);
  const [selectedTimeRange, setSelectedTimeRange] = useState<TimeRange>(defaultTimeRange);
  const environment = useEnvironment();

  const [{ data }] = useQuery({
    query: GetFunctionRunsCountDocument,
    variables: {
      environmentID: environment.id,
      functionSlug,
      functionRunStatuses: selectedStatuses.length ? selectedStatuses : null,
      timeRangeStart: selectedTimeRange.start.toISOString(),
      timeRangeEnd: selectedTimeRange.end.toISOString(),
      timeField: selectedTimeField,
    },
  });
  const { user } = useUser();
  const { organization } = useOrganization();

  const functionRunsCount = data?.environment.function?.runs?.totalCount;

  function handleStatusesChange(statuses: FunctionRunStatus[]) {
    setSelectedStatuses(statuses);
    window.inngest.send({
      name: 'app/filter.selected',
      data: {
        list: 'function-runs',
        type: 'status',
        value: statuses,
      },
      ...(user &&
        organization && {
          user: {
            external_id: user.externalId,
            email: user.primaryEmailAddress?.emailAddress,
            name: user.fullName,
            account_id: organization.publicMetadata.accountID,
          },
        }),
      v: '2023-06-02.1',
    });
  }

  function handleTimeRangeChange(timeRange: TimeRange) {
    setSelectedTimeRange(timeRange);
    window.inngest.send({
      name: 'app/filter.selected',
      data: {
        list: 'function-runs',
        type: 'time-range',
        value: timeRange,
      },
      ...(user &&
        organization && {
          user: {
            external_id: user.externalId,
            email: user.primaryEmailAddress?.emailAddress,
            name: user.fullName,
            account_id: organization.publicMetadata.accountID,
          },
        }),
      v: '2023-06-05.1',
    });
  }

  return (
    <>
      <div className="flex items-center justify-between gap-2 border-b border-slate-300 px-5 py-2">
        <div className="gap flex items-center gap-1.5">
          <StatusFilter
            selectedStatuses={selectedStatuses}
            onStatusesChange={handleStatusesChange}
          />
          <TimeRangeFilter
            selectedTimeField={selectedTimeField}
            selectedTimeRange={selectedTimeRange}
            onTimeFieldChange={setSelectedTimeField}
            onTimeRangeChange={handleTimeRangeChange}
          />
          {functionRunsCount !== undefined && (
            <p className="text-sm font-semibold text-slate-900">{functionRunsCount} Runs</p>
          )}
        </div>
        <ClientFeatureFlag flag="function-replay">
          <NewReplayButton functionSlug={functionSlug} />
        </ClientFeatureFlag>
      </div>
      <div className="flex min-h-0 flex-1">
        <FunctionRunList
          functionSlug={functionSlug}
          selectedStatuses={selectedStatuses}
          selectedTimeRange={selectedTimeRange}
          timeField={selectedTimeField}
        />
      </div>
    </>
  );
}
