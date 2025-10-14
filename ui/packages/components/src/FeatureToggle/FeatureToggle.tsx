import { useEffect, useState } from 'react';

import { FEATURE_FLAG_NAMESPACE, useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { Switch } from '../Switch';
import { cn } from '../utils/classNames';

type featureToggleProps = {
  featureFlagName: string;
  defaultEnabled?: boolean;
  className?: string;
};

export const FeatureToggle = ({
  featureFlagName,
  defaultEnabled = false,
  className,
}: featureToggleProps) => {
  const storageKey = `${FEATURE_FLAG_NAMESPACE}${featureFlagName}`;
  const rawFlag = typeof window === 'undefined' ? null : window.localStorage.getItem(storageKey);
  const localStorageEnabled = rawFlag === null ? null : rawFlag === 'true';
  const [enabled, setEnabled] = useState<boolean | null>(localStorageEnabled);

  const { booleanFlag } = useBooleanFlag();
  const { value: featureFlagEnabled, isReady: featureFlagReady } = booleanFlag(
    featureFlagName,
    defaultEnabled
  );

  useEffect(() => {
    //
    // localstorage value takes precedence
    if (enabled === null) {
      setEnabled(featureFlagEnabled);
    }
  }, [featureFlagReady, featureFlagEnabled]);

  return (
    <Switch
      id={storageKey}
      checked={localStorageEnabled !== null ? localStorageEnabled : featureFlagEnabled}
      className={cn('data-[state=checked]:bg-primary-moderate cursor-pointer', className)}
      onClick={(e) => {
        e.stopPropagation();
        localStorage.setItem(storageKey, JSON.stringify(!enabled));
        setEnabled(!enabled);
      }}
    />
  );
};
