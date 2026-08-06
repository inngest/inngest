import { useCallback } from 'react';

import { useTracking } from '@/hooks/useTracking';

export type FakeDoorFeature = 'scores' | 'experiments';

export type FakeDoorAction =
  | 'viewed'
  | 'prompt_copied'
  | 'example_copied'
  | 'docs_clicked'
  | 'cta_clicked';

// Tracks interactions with the fake-door Scores/Experiments pages via a
// single event, so reach and click-through can be compared against the
// cli/dev_ui.loaded baseline in one funnel.
export function useFakeDoorTracking(feature: FakeDoorFeature) {
  const { trackEvent } = useTracking();

  return useCallback(
    (action: FakeDoorAction) => {
      trackEvent('cli/dev_ui.fake_door.action', { feature, action });
    },
    [feature, trackEvent],
  );
}
