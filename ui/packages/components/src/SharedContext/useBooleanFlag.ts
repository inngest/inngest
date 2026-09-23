import { useShared } from './SharedContext';
import type { ClientFeatureFlagKey } from './clientFeatureFlags';

export type BooleanFlag = {
  // Ready means the value can be used, including the default after an error.
  // False means flag identification is still pending.
  isReady: boolean;

  value: boolean;
};

export type BooleanFlagPayload = {
  flag: ClientFeatureFlagKey;
  defaultValue: boolean;
  overrideable?: boolean;
};

export const FEATURE_FLAG_NAMESPACE = 'inngest-feature-flag-';

export const useBooleanFlag = () => {
  const shared = useShared();
  const booleanFlag = (
    flag: ClientFeatureFlagKey,
    defaultValue: boolean = false,
    userOverrideable: boolean = false
  ): BooleanFlag => {
    if (userOverrideable && typeof window !== 'undefined') {
      const localStorageEnabled = localStorage.getItem(`${FEATURE_FLAG_NAMESPACE}${flag}`);
      if (localStorageEnabled !== null) {
        return { isReady: true, value: localStorageEnabled === 'true' };
      }
    }
    return shared.booleanFlag(flag, defaultValue);
  };

  return {
    booleanFlag,
  };
};
