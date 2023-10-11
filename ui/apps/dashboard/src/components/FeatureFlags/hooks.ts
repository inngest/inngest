import { useContext } from 'react';
import { useFlags } from 'launchdarkly-react-client-sdk';

import { IdentificationContext } from './ClientFeatureFlagProvider';

type BooleanFlag = {
  // Whether the flag is ready to be used. This will be false if the user has
  // not been identified in the LaunchDarkly client.
  isReady: boolean;

  value: boolean;
};

export function useBooleanFlag(flag: string, defaultValue: boolean = false): BooleanFlag {
  const value: unknown = useFlags()[flag];
  const isIdentified = useIsIdentified();

  if (!isIdentified) {
    return { isReady: false, value: defaultValue };
  }

  if (typeof value === 'undefined') {
    console.error(`flag ${flag} is not available`);
    return { isReady: false, value: defaultValue };
  }

  if (typeof value !== 'boolean') {
    console.error(`flag ${flag} is not a boolean`);
    return { isReady: false, value: defaultValue };
  }

  return { isReady: true, value };
}

/**
 * Returns true if the user is identified in the LaunchDarkly client. This is
 * useful when you want to ensure that the user is identified before querying
 * for flags.
 */
function useIsIdentified(): boolean {
  return useContext(IdentificationContext).isIdentified;
}
