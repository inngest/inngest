import { useOrganization } from '@clerk/tanstack-react-start';
import { useCallback } from 'react';

import {
  trackAssistantActionPerformed,
  type AssistantAction,
} from '@/utils/analyticsEvents';

export type RunAIAction = AssistantAction;

export function useRunAITracking(
  scope: 'env' | 'fn' | 'debug',
  detailView?: 'inline' | 'standalone',
) {
  const { organization } = useOrganization();
  const accountId = organization?.publicMetadata.accountID;
  return useCallback(
    (event: RunAIAction) => {
      try {
        void Promise.resolve(
          trackAssistantActionPerformed({
            feature: 'runs-ai-assist',
            accountId: typeof accountId === 'string' ? accountId : undefined,
            event,
            surface:
              scope === 'debug'
                ? 'run_details'
                : scope === 'fn'
                ? 'function_runs'
                : 'environment_runs',
            templateVersion: scope === 'debug' ? 'run_debug_v1' : 'basic_v1',
            ...(scope === 'debug' ? { detailView } : {}),
          }),
        ).catch(() => {});
      } catch {
        // Analytics must not interrupt user actions.
      }
    },
    [scope, detailView, accountId],
  );
}
