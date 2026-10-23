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

export function useRunAITracking(
  scope: 'env' | 'fn' | 'debug',
  detailView?: 'inline' | 'standalone',
) {
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
            surface:
              scope === 'debug'
                ? 'run_details'
                : scope === 'fn'
                  ? 'function_runs'
                  : 'environment_runs',
            template_version: scope === 'debug' ? 'run_debug_v1' : 'basic_v1',
            ...(scope === 'debug' ? { detail_view: detailView } : {}),
          },
        });
      } catch {
        // Analytics must not interrupt user actions.
      }
    },
    [scope, detailView, user],
  );
}
