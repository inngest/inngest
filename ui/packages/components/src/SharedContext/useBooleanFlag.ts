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
};

export const useBooleanFlag = () => {
  const shared = useShared();
  const booleanFlag = (flag: string, defaultValue: boolean = false): BooleanFlag =>
    shared.booleanFlag(flag, defaultValue);

  return {
    booleanFlag,
  };
};
