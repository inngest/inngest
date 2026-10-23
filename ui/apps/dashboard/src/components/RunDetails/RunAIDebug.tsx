import { useOrganization } from '@clerk/tanstack-react-start';
import type { GetRunData } from '@inngest/components/SharedContext/useGetRun';

import { useEnvironment } from '@/components/Environments/environment-context';
import { RunAIAssistPopover } from '@/components/Runs/AIAssist/RunAIAssistPopover';
import { useRunAITracking } from '@/components/Runs/AIAssist/tracking';
import { useEnvironments } from '@/queries/environments';

export function RunAIDebug({
  run,
  standalone,
}: {
  run: GetRunData;
  standalone: boolean;
}) {
  const { organization } = useOrganization();
  const environment = useEnvironment();
  const [{ data, fetching, error }, refetch] = useEnvironments();
  const track = useRunAITracking('debug', standalone ? 'standalone' : 'inline');
  const canonicalEnvironment = data?.find((env) => env.id === environment.id);
  const accountID = organization?.publicMetadata.accountID;

  if (!organization || run.status !== 'FAILED') return null;

  return (
    <RunAIAssistPopover
      key={`${organization.id}:${environment.id}:${run.id}`}
      mode="debug"
      context={{
        accountID: typeof accountID === 'string' ? accountID : '',
        accountName: organization.name,
        environmentSlug: canonicalEnvironment?.slug ?? '',
        environmentLabel: environment.slug,
        runID: run.id,
        functionName: run.fn.name,
      }}
      endpoint={new URL('/mcp', import.meta.env.VITE_API_URL).toString()}
      metadataLoading={fetching}
      metadataError={Boolean(error) || (!fetching && !canonicalEnvironment)}
      onRetry={() => refetch({ requestPolicy: 'network-only' })}
      onAction={track}
    />
  );
}

export const renderRunAIDebug = (run: GetRunData, standalone: boolean) => (
  <RunAIDebug run={run} standalone={standalone} />
);
