import type { ClientFeatureFlagKey } from '@inngest/components/SharedContext/clientFeatureFlags';

import { useBooleanFlag } from './hooks';

type Props = {
  children: React.ReactNode;
  defaultValue?: boolean;
  flag: ClientFeatureFlagKey;
};

// Conditionally renders children based on a feature flag.
export function ClientFeatureFlag({
  children,
  defaultValue = false,
  flag,
}: Props) {
  const { value: isEnabled } = useBooleanFlag(flag, defaultValue);

  if (isEnabled) {
    return <>{children}</>;
  }
  return <></>;
}
