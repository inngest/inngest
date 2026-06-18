import { useEffect, useState } from 'react';

const NAV_OVERRIDE_KEY = 'navVersion';
const NAV_QUERY_PARAM = 'nav';

type NavOverride = 'v1' | 'v2' | null;

type NavigationV2State = {
  value: boolean;
  isReady: boolean;
};

function parseOverride(value: string | null): NavOverride {
  return value === 'v1' || value === 'v2' ? value : null;
}

// Reads a manual navigation-version override from localStorage, persisting one
// first if a `?nav=v1|v2` query param is present. The override takes precedence
// over the default (V2), so once you opt in via the query param you keep seeing
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

// The redesigned (V2) navigation is now the default for all accounts. A manual
// `?nav=v1` override (see readNavOverride) still lets a user fall back to the
// old navigation as a safety valve; passing `?nav=v2` returns to the default.
export function useNavigationV2State(): NavigationV2State {
  // Resolved after mount to stay SSR/hydration-safe — no window access during
  // the first render, so the initial markup matches the default (V2).
  const [override, setOverride] = useState<NavOverride | undefined>(undefined);
  useEffect(() => {
    setOverride(readNavOverride());
  }, []);

  if (override === undefined) {
    return { isReady: false, value: true };
  }

  if (override) {
    return { isReady: true, value: override === 'v2' };
  }

  return { isReady: true, value: true };
}

export function useNavigationV2(): boolean {
  return useNavigationV2State().value;
}
