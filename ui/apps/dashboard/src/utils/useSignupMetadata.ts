import { useEffect, useState } from 'react';

import { getSignupMetadata } from './signupAttribution';

// On app.inngest.com, Segment loads GTM (and with it GA4 and the Conversion
// Linker) after the page hydrates. On a visitor's first page view, the GA
// cookies can therefore appear after Clerk's form has mounted, so keep
// re-reading briefly; Clerk applies prop updates to the mounted form without
// remounting it, and sends whatever metadata is current at submit/OAuth time.
const REFRESH_INTERVAL_MS = 1000;
const MAX_REFRESHES = 20;

const isSameRecord = (a: Record<string, string>, b: Record<string, string>) => {
  const aKeys = Object.keys(a);
  return (
    aKeys.length === Object.keys(b).length &&
    aKeys.every((key) => a[key] === b[key])
  );
};

export const useSignupMetadata = (): Record<string, string> => {
  const [metadata, setMetadata] = useState(getSignupMetadata);

  useEffect(() => {
    const refresh = (): boolean => {
      const next = getSignupMetadata();
      setMetadata((prev) => (isSameRecord(prev, next) ? prev : next));
      return Boolean(next.gaClientId && next.gaSessionId);
    };

    if (refresh()) return;

    let refreshes = 0;
    const timer = window.setInterval(() => {
      refreshes += 1;
      if (refresh() || refreshes >= MAX_REFRESHES) window.clearInterval(timer);
    }, REFRESH_INTERVAL_MS);

    return () => window.clearInterval(timer);
  }, []);

  return metadata;
};
