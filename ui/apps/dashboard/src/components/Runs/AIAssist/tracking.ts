import { useCallback } from 'react';

import { trackEvent, useTrackingUser } from '@/utils/tracking';

export type RunAIAction =
  | { action: 'viewed' | 'opened' | 'prompt_copy_clicked' | 'prompt_copied' }
  | {
      action: 'setup_clicked' | 'setup_copied' | 'setup_copy_failed';
      client: 'claude_code' | 'codex_cli' | 'other';
    }
  | {
      action: 'prompt_copy_failed';
      failure_category: 'permission_denied' | 'unavailable' | 'other';
    }
  | {
      action: 'guide_clicked';
      guide: 'connection' | 'troubleshooting' | 'more_ai_resources';
    };

export function useRunAITracking(scope: 'env' | 'fn') {
  const user = useTrackingUser();
  return useCallback(
    (event: RunAIAction) => {
      try {
        trackEvent({
          name: 'app/runs.ai_assist.action',
          v: '1',
          user,
          data: {
            ...event,
            surface: scope === 'fn' ? 'function_runs' : 'environment_runs',
            template_version: 'basic_v1',
          },
        });
      } catch {
        // Analytics must not interrupt user actions.
      }
    },
    [scope, user],
  );
}
