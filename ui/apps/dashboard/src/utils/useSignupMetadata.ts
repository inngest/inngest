import { useEffect, useState } from 'react';

import { getSignupMetadata } from './signupAttribution';

// On app.inngest.com, Segment loads GTM (and with it GA4 and the Conversion
// Linker) after the page hydrates. On a visitor's first page view, the GA
// cookies and the Conversion Linker's click-ID cookie can therefore appear
// after Clerk's form has mounted, and not necessarily together, so keep
// re-reading briefly; Clerk applies prop updates to the mounted form without
// remounting it, and sends whatever metadata is current at submit/OAuth time.
// Polling stops early only once both GA IDs and a click ID are present;
// visitors who arrive without an ad click simply poll until the cap.
const REFRESH_INTERVAL_MS = 1000;
const MAX_REFRESHES = 20;

const isSameRecord = (a: Record<string, string>, b: Record<string, string>) => {
  const aKeys = Object.keys(a);
  return (
    aKeys.length === Object.keys(b).length &&
    aKeys.every((key) => a[key] === b[key])
  );
};

const isComplete = (metadata: Record<string, string>): boolean =>
  Boolean(
    metadata.gaClientId &&
      metadata.gaSessionId &&
      (metadata.gclid || metadata.gbraid || metadata.wbraid),
  );

export const useSignupMetadata = (): Record<string, string> => {
  const [metadata, setMetadata] = useState(getSignupMetadata);

  useEffect(() => {
    const refresh = (): boolean => {
      const next = getSignupMetadata();
      setMetadata((prev) => (isSameRecord(prev, next) ? prev : next));
      return isComplete(next);
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
