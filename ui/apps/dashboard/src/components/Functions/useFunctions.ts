import { useCallback } from 'react';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { GetFunctionUsageDocument, GetFunctionsDocument } from '@/gql/graphql';

import { concurrencyLimitReachedBySlot } from './concurrency';

type QueryVariables = {
  archived: boolean;
  nameSearch: string | null;
  appIDs: string[] | null;
  cursor: number | null;
};

export function useFunctions() {
  const envID = useEnvironment().id;
  const client = useClient();
  return useCallback(
    async ({ cursor, archived, nameSearch, appIDs }: QueryVariables) => {
      const result = await client
        .query(
          GetFunctionsDocument,
          {
            environmentID: envID,
            page: cursor ?? 1,
            pageSize: 50,
            archived,
            search: nameSearch,
            appIDs,
          },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const { page, data } = result.data.workspace.workflows;
      const flattenedFunctions = data.map((fn) => ({
        ...fn,
        triggers: fn.triggers,
      }));

      return {
        functions: flattenedFunctions,
        pageInfo: {
          currentPage: page.page,
          totalPages: page.totalPages,
        },
      };
    },
    [client, envID],
  );
}

export function useFunctionVolume() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ functionID }: { functionID: string }) => {
      const startTime = getTimestampDaysAgo({
        currentDate: new Date(),
        days: 1,
      }).toISOString();
      const endTime = new Date().toISOString();

      const result = await client
        .query(
          GetFunctionUsageDocument,
          {
            id: functionID,
            environmentID: envID,
            startTime,
            endTime,
          },
          { requestPolicy: 'network-only' },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error('no data returned');
      }

      const workflow = result.data.workspace.workflow;

      if (!workflow) {
        throw new Error('function not found');
      }

      // Calculate totals
      const dailyFailureCount = workflow.dailyFailures.total;
      const dailyFinishedCount =
        workflow.dailyCompleted.total +
        workflow.dailyCancelled.total +
        dailyFailureCount;

      // Calculate failure rate percentage (rounded to 2 decimal places)
      const failureRate = dailyFinishedCount
        ? Math.round((dailyFailureCount / dailyFinishedCount) * 10000) / 100
        : 0;

      const concurrencyLimitReached = concurrencyLimitReachedBySlot(
        workflow.dailyStarts.data,
        workflow.concurrencyLimitReached.data,
      );

      // Creates an array of objects containing the start and failure count for each usage slot.
      const dailyVolumeSlots = workflow.dailyStarts.data.map(
        (usageSlot, index) => ({
          concurrencyLimitReached: concurrencyLimitReached[index] ?? false,
          startCount: usageSlot.count,
          failureCount: workflow.dailyFailures.data[index]?.count ?? 0,
        }),
      );

      const usage = {
        dailyVolumeSlots,
        totalVolume: workflow.dailyStarts.total,
      };

      return {
        failureRate,
        usage,
      };
    },
    [client, envID],
  );
}
