import { useCallback, useRef } from 'react';

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

export function getRunAIVisitID(search: string): string {
  const existing = new URLSearchParams(search).get('aiVisit');
  return existing && /^[a-f0-9-]{36}$/i.test(existing)
    ? existing
    : crypto.randomUUID();
}

export function useRunAITracking(scope: 'env' | 'fn', search: string) {
  const user = useTrackingUser();
  const visitID = useRef<string>();
  const track = useCallback(
    (event: RunAIAction) => {
      if (!visitID.current) visitID.current = getRunAIVisitID(search);
      let alreadyViewed = false;
      if (event.action === 'viewed') {
        try {
          alreadyViewed =
            sessionStorage.getItem(`runs-ai-viewed:${visitID.current}`) ===
            'true';
        } catch {
          /* Storage may be unavailable; in-component deduplication still applies. */
        }
      }
      if (alreadyViewed) return;
      try {
        trackEvent({
          name: 'app/runs.ai_assist.action',
          v: '1',
          user,
          data: {
            ...event,
            surface: scope === 'fn' ? 'function_runs' : 'environment_runs',
            template_version: 'basic_v1',
            visit_id: visitID.current,
          },
        });
        if (event.action === 'viewed') {
          try {
            sessionStorage.setItem(`runs-ai-viewed:${visitID.current}`, 'true');
          } catch {
            /* Tracking also works without session storage. */
          }
        }
      } catch {
        // Analytics must never interrupt copying or setup navigation.
      }
    },
    [scope, search, user],
  );
  return {
    track,
    getVisitID: () => {
      if (!visitID.current) visitID.current = getRunAIVisitID(search);
      return visitID.current;
    },
  };
}
