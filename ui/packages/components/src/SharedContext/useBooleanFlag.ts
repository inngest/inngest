import { useShared } from './SharedContext';

export type BooleanFlag = {
  // Whether the flag is ready to be used. This will be false if the user has
  // not been identified in the LaunchDarkly client.
  isReady: boolean;

  value: boolean;
};

export type BooleanFlagPayload = {
  flag: string;
  defaultValue: boolean;
  overrideable?: boolean;
};

export const FEATURE_FLAG_NAMESPACE = 'inngest-feature-flag-';

export const useBooleanFlag = () => {
  const shared = useShared();
  const booleanFlag = (
    flag: string,
    defaultValue: boolean = false,
    userOverrideable: boolean = false
  ): BooleanFlag => {
    if (userOverrideable) {
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
