import { useMemo } from 'react';
import { Button } from '@inngest/components/Button/Button';
import { FunctionsTable } from '@inngest/components/Functions/FunctionsTable';
import { Header } from '@inngest/components/Header/Header';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';
import { createFileRoute, getRouteApi, useNavigate } from '@tanstack/react-router';
import { useClient } from 'urql';

import { FunctionInfo } from '@/components/Functions/FunctionInfo';
import { GetFunctionUsageDocument, GetFunctionsDocument } from '@/gql/graphql';
import { useEnvironmentContext } from '@/spa/contexts/EnvironmentContext';

// Create route API for type-safe hooks (with CLI limitations workaround)
const routeApi = getRouteApi('/env/$envSlug/functions' as any);

// Create path creator for TanStack Router
function createTanStackPathCreator(envSlug: string) {
  return {
    function: (params: { functionSlug: string }) =>
      `/env/${envSlug}/functions/${params.functionSlug}`,
    eventType: (params: { eventName: string }) => `/env/${envSlug}/event-types/${params.eventName}`,
    app: (params: { externalAppID: string }) =>
      `/env/${envSlug}/apps/${encodeURIComponent(params.externalAppID)}`,
  };
}

// Functions hook adapted for TanStack Router
function useTanStackFunctions(envID: string) {
  const client = useClient();

  return useMemo(() => {
    return async ({
      cursor,
      archived,
      nameSearch,
    }: {
      archived: boolean;
      nameSearch: string | null;
      cursor: number | null;
    }) => {
      const result = await client
        .query(
          GetFunctionsDocument,
          {
            environmentID: envID,
            page: cursor ?? 1,
            pageSize: 30,
            archived,
            search: nameSearch,
          },
          { requestPolicy: 'network-only' }
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
        triggers: fn.current?.triggers ?? [],
      }));

      return {
        functions: flattenedFunctions,
        pageInfo: {
          currentPage: page.page,
          totalPages: page.totalPages,
        },
      };
    };
  }, [client, envID]);
}

// Function volume hook adapted for TanStack Router
function useTanStackFunctionVolume(envID: string) {
  const client = useClient();

  return useMemo(() => {
    return async ({ functionID }: { functionID: string }) => {
      const startTime = getTimestampDaysAgo({ currentDate: new Date(), days: 1 }).toISOString();
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
          { requestPolicy: 'network-only' }
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
        workflow.dailyCompleted.total + workflow.dailyCancelled.total + dailyFailureCount;

      // Calculate failure rate percentage (rounded to 2 decimal places)
      const failureRate = dailyFinishedCount
        ? Math.round((dailyFailureCount / dailyFinishedCount) * 10000) / 100
        : 0;

      // Creates an array of objects containing the start and failure count for each usage slot (1 hour)
      const dailyVolumeSlots = workflow.dailyStarts.data.map((usageSlot, index) => ({
        startCount: usageSlot.count,
        failureCount: workflow.dailyFailures.data[index]?.count ?? 0,
      }));

      const usage = {
        dailyVolumeSlots,
        totalVolume: dailyFinishedCount,
      };

      return {
        failureRate,
        usage,
      };
    };
  }, [client, envID]);
}

function FunctionsPage() {
  const navigate = useNavigate();
  const { envSlug } = (routeApi as any).useParams();
  const { environment, loading, error } = useEnvironmentContext();

  const pathCreator = useMemo(() => createTanStackPathCreator(envSlug), [envSlug]);
  const getFunctions = useTanStackFunctions(environment?.id || '');
  const getFunctionVolume = useTanStackFunctionVolume(environment?.id || '');

  if (loading || !environment) {
    return (
      <div className="mt-16 flex place-content-center">
        <div className="rounded-lg border border-blue-200 bg-blue-50 p-6 text-center">
          <h2 className="text-lg font-semibold text-blue-900">Loading functions...</h2>
          <p className="mt-2 text-sm text-gray-600">
            {loading ? 'Loading environment data...' : 'Waiting for environment data to load.'}
          </p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="mt-16 flex place-content-center">
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-center">
          <h2 className="text-lg font-semibold text-red-900">Failed to load environment</h2>
          <p className="mt-2 text-sm text-gray-600">{error}</p>
        </div>
      </div>
    );
  }

  const handleRefresh = () => {
    // In TanStack Router, we can invalidate the route to refresh data
    navigate({ to: `/env/$envSlug/functions`, params: { envSlug } } as any);
  };

  return (
    <>
      <Header breadcrumb={[{ text: 'Functions' }]} infoIcon={<FunctionInfo />} />
      <FunctionsTable
        pathCreator={pathCreator}
        getFunctions={getFunctions}
        getFunctionVolume={getFunctionVolume}
        emptyActions={
          <>
            <Button
              appearance="outlined"
              label="Refresh"
              onClick={handleRefresh}
              icon={<RiRefreshLine />}
              iconSide="left"
            />
            <Button
              label="Go to docs"
              href="https://www.inngest.com/docs/learn/inngest-functions"
              target="_blank"
              icon={<RiExternalLinkLine />}
              iconSide="left"
            />
          </>
        }
      />
    </>
  );
}

export const Route = createFileRoute('/env/$envSlug/functions')({
  component: FunctionsPage,
});
