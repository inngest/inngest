import { useOrganization } from '@clerk/tanstack-react-start';
import { useLocation } from '@tanstack/react-router';

import { useEnvironments } from '@/queries/environments';

import type { RunPromptContext } from './prompt';
import { RunAIAssistPopover } from './RunAIAssistPopover';
import { useRunAITracking } from './tracking';

type Props = Omit<RunPromptContext, 'account' | 'search'> & {
  environmentID: string;
  metadataLoading: boolean;
  metadataError: boolean;
  onRetry: () => void;
};

export function RunAIAssist(props: Props) {
  const { organization } = useOrganization();
  const location = useLocation();
  const track = useRunAITracking(props.scope);
  const [{ data: environments, fetching, error }, refetchEnvironments] =
    useEnvironments();
  // Dashboard aliases (such as "production") may differ from MCP environment IDs.
  const apiEnvironment = environments?.find(
    (env) => env.id === props.environmentID,
  );
  // Clerk is absent in self-hosted mode. The local Runs page never supplies this slot.
  if (!organization) return null;

  const context = {
    ...props,
    account: organization.name,
    environmentSlug: apiEnvironment?.slug ?? '',
    environmentLabel: props.environmentSlug,
    search: location.search,
  };
  return (
    <RunAIAssistPopover
      context={context}
      endpoint={new URL('/mcp', import.meta.env.VITE_API_URL).toString()}
      metadataLoading={props.metadataLoading || fetching}
      metadataError={
        props.metadataError || Boolean(error) || (!fetching && !apiEnvironment)
      }
      onRetry={() => {
        props.onRetry();
        refetchEnvironments({ requestPolicy: 'network-only' });
      }}
      onAction={track}
    />
  );
}
