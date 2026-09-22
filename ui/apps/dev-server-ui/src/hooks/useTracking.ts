import { useCallback } from 'react';
import { createDevServerURL } from '@/utils/devServer';

export function useTracking() {
  // Stable across renders so callers can safely list trackEvent in their
  // own hook dependencies without re-triggering effects.
  const trackEvent = useCallback(
    async (eventName: string, eventData: Record<string, any> = {}) => {
      try {
        const response = await fetch(createDevServerURL('/v0/telemetry'), {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            eventName,
            ...eventData,
          }),
        });

        if (!response.ok) {
          console.error('Failed to send telemetry event');
        }
      } catch (err) {
        console.error(
          err instanceof Error ? err : new Error('An error occurred'),
        );
      }
    },
    [],
  );

  return { trackEvent };
}
