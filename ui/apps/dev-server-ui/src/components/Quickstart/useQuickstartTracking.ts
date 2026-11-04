import { useCallback, useRef } from 'react';
import { useTracking } from '@/hooks/useTracking';
import { storageKey } from './state';

type Action =
  | 'viewed'
  | 'agent_prompt_copied'
  | 'manual_install_copy_clicked'
  | 'docs_clicked'
  | 'app_ready'
  | 'invoke_clicked'
  | 'first_run_completed';
type Properties = {
  path?: 'agent' | 'manual';
  entry_stage?: 'setup' | 'run' | 'complete';
  guide?: 'mcp' | 'manual' | 'overview';
  language?: string;
  package_manager?: string;
};

// Use the existing CLI telemetry pipeline; no new identity or analytics client.
export function useQuickstartTracking(server: string) {
  const { trackEvent } = useTracking();
  const sent = useRef(new Set<string>());
  return useCallback(
    (action: Action, properties: Properties = {}, once = false) => {
      if (!server) return;
      if (once) {
        const key = `${storageKey(server)}:tracking:${action}`;
        if (sent.current.has(key)) return;
        try {
          if (sessionStorage.getItem(key)) return;
          sessionStorage.setItem(key, '1');
        } catch {
          // The current mount still deduplicates when browser storage is blocked.
        }
        sent.current.add(key);
      }
      void trackEvent('cli/onboarding.action', {
        flow: 'quickstart',
        version: 1,
        action,
        ...properties,
      });
    },
    [server, trackEvent],
  );
}
