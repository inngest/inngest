import { useOrganization } from '@clerk/tanstack-react-start';
import { useLocation } from '@tanstack/react-router';

import type { RunPromptContext } from './prompt';
import { RunAIAssistPopover } from './RunAIAssistPopover';
import { runAISetupHref } from './setup';
import { useRunAITracking } from './tracking';

type Props = Omit<RunPromptContext, 'account' | 'search'> & {
  metadataLoading: boolean;
  metadataError: boolean;
  onRetry: () => void;
};

export function RunAIAssist(props: Props) {
  const { organization } = useOrganization();
  const location = useLocation();
  const { track, getVisitID } = useRunAITracking(
    props.scope,
    location.searchStr,
  );
  // Clerk is absent in self-hosted mode. The local Runs page never supplies this slot.
  if (!organization) return null;

  const context = {
    ...props,
    account: organization.name,
    search: location.search,
  };
  return (
    <RunAIAssistPopover
      context={context}
      endpoint={new URL('/mcp', import.meta.env.VITE_API_URL).toString()}
      metadataLoading={props.metadataLoading}
      metadataError={props.metadataError}
      onRetry={props.onRetry}
      onAction={track}
      getSetupHref={(client) => {
        const search = new URLSearchParams(location.searchStr);
        search.set('aiVisit', getVisitID());
        return runAISetupHref(
          client,
          `${location.pathname}?${search.toString()}`,
        );
      }}
    />
  );
}
