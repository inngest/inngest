import { useEffect, useState } from 'react';

import { useShared, type SharedDefinitions } from './SharedContext';

export type LegacyTraceType = {
  enabled: boolean;
  toggle: () => void;
  ready: boolean;
};

const key = 'LegacyTraces:enabled';

export const useLegacyTrace = (): LegacyTraceType => {
  const shared = useShared();
  return shared.legacyTrace;
};

export const legacyTraceToggle = () => {
  const [enabled, setEnabled] = useState(false);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const enabled = window.localStorage.getItem(key);
    setEnabled(enabled === 'true');
    setReady(true);
  }, []);

  const toggle = () => {
    const toggled = !enabled;
    window.localStorage.setItem(key, JSON.stringify(toggled));
    setEnabled(toggled);
  };

  return { ready, enabled, toggle };
};
