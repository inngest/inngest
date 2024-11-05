import { useEffect, useState } from 'react';

import { STEPS_ORDER, type OnboardingSteps, type TotalStepsCompleted } from './types';
import { useOnboardingTracking } from './useOnboardingTracking';

export default function useOnboardingStep() {
  // Temporary approach, we will store this value in the backend in the future
  const [lastCompletedStep, setLastCompletedStep] = useState<OnboardingSteps | undefined>(
    undefined
  );

  const tracking = useOnboardingTracking();

  useEffect(() => {
    const stored = localStorage.getItem('onboardingLastStepCompleted');
    if (stored) {
      setLastCompletedStep(JSON.parse(stored));
    }

    const handleStorageChange = (event: StorageEvent) => {
      if (event.key === 'onboardingLastStepCompleted') {
        const newValue = event.newValue ? JSON.parse(event.newValue) : undefined;
        setLastCompletedStep(newValue);
      }
    };

    // Listen for storage events from other components
    window.addEventListener('storage', handleStorageChange);

    // Custom event for same-window updates
    const handleCustomEvent = (event: CustomEvent) => {
      setLastCompletedStep(event.detail);
    };

    window.addEventListener('onboardingStepUpdate', handleCustomEvent as EventListener);

    return () => {
      window.removeEventListener('storage', handleStorageChange);
      window.removeEventListener('onboardingStepUpdate', handleCustomEvent as EventListener);
    };
  }, []);

  const isFinalStep = lastCompletedStep === STEPS_ORDER[STEPS_ORDER.length - 1];
  const nextStep = (
    !lastCompletedStep
      ? STEPS_ORDER[0]
      : isFinalStep
      ? lastCompletedStep
      : STEPS_ORDER[STEPS_ORDER.indexOf(lastCompletedStep) + 1]
  ) as OnboardingSteps;

  const totalStepsCompleted: TotalStepsCompleted = isFinalStep
    ? STEPS_ORDER.length
    : STEPS_ORDER.indexOf(nextStep);

  const updateLastCompletedStep = (
    step: OnboardingSteps,
    completionSource: string = 'automatic'
  ) => {
    if (typeof window !== 'undefined') {
      // Update localStorage
      localStorage.setItem('onboardingLastStepCompleted', JSON.stringify(step));

      // Update local state
      setLastCompletedStep(step);

      // Dispatch custom event for other components in the same window
      window.dispatchEvent(new CustomEvent('onboardingStepUpdate', { detail: step }));

      const willBeFinalStep = step === STEPS_ORDER[STEPS_ORDER.length - 1];
      tracking?.trackOnboardingStepCompleted(step, willBeFinalStep, completionSource);
    }
  };

  return { lastCompletedStep, updateLastCompletedStep, isFinalStep, nextStep, totalStepsCompleted };
}
