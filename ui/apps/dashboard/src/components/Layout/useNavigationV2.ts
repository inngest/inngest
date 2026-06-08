import { useEffect, useState } from 'react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';

export const NAVIGATION_V2_FLAG = 'navigation-v2';

const NAV_OVERRIDE_KEY = 'navVersion';
const NAV_QUERY_PARAM = 'nav';

type NavOverride = 'v1' | 'v2' | null;

function parseOverride(value: string | null): NavOverride {
  return value === 'v1' || value === 'v2' ? value : null;
}

// Reads a manual navigation-version override from localStorage, persisting one
// first if a `?nav=v1|v2` query param is present. The override always wins over
// the LaunchDarkly flag, so once you opt in via the query param you keep seeing
// that version on subsequent visits (until you pass the other value).
function readNavOverride(): NavOverride {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const param = parseOverride(
      new URLSearchParams(window.location.search).get(NAV_QUERY_PARAM),
    );
    if (param) {
      window.localStorage.setItem(NAV_OVERRIDE_KEY, param);
      return param;
    }
    return parseOverride(window.localStorage.getItem(NAV_OVERRIDE_KEY));
  } catch {
    return null;
  }
}

// Whether the redesigned (V2) navigation is enabled for the current account.
// Defaults to false (old navigation) until LaunchDarkly identifies the account,
// so accounts not targeted for the flag always render the old nav. A manual
// `?nav=v1|v2` override (see readNavOverride) takes precedence over the flag.
export function useNavigationV2(): boolean {
  const { value: flagValue } = useBooleanFlag(NAVIGATION_V2_FLAG);

  // Resolved after mount to stay SSR/hydration-safe — no window access during
  // the first render, so the initial markup matches the flag-driven default.
  const [override, setOverride] = useState<NavOverride>(null);
  useEffect(() => {
    setOverride(readNavOverride());
  }, []);

  if (override) {
    return override === 'v2';
  }

  return flagValue;
}
