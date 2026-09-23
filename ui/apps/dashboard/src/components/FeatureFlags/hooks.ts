import { useContext, useEffect } from 'react';
import type { ClientFeatureFlagKey } from '@inngest/components/SharedContext/clientFeatureFlags';
import type { BooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';
import { useFlags } from 'launchdarkly-react-client-sdk';

import { IdentificationContext } from './ClientFeatureFlagProvider';

export function useBooleanFlag(
  flag: ClientFeatureFlagKey,
  defaultValue: boolean = false,
): BooleanFlag {
  const value: unknown = useFlags()[flag];
  const { isIdentified, hasError } = useContext(IdentificationContext);
  const failure = hasError
    ? 'feature flag initialization or identification failed'
    : !isIdentified
    ? undefined
    : value === undefined
    ? 'flag unavailable'
    : typeof value !== 'boolean'
    ? 'expected a boolean value'
    : undefined;

  useEffect(() => {
    if (failure) {
      console.error(
        `Couldn't load flag "${flag}": ${failure}; using default ${defaultValue}`,
      );
    }
  }, [flag, failure, defaultValue]);

  if (failure) return { isReady: true, value: defaultValue };
  if (!isIdentified) return { isReady: false, value: defaultValue };
  return { isReady: true, value: value as boolean };
}
