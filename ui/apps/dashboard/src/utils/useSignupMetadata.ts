import { useEffect, useState } from 'react';

import { getSignupMetadata } from './signupAttribution';

// GTM can write GA and click cookies after the auth form mounts.
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
