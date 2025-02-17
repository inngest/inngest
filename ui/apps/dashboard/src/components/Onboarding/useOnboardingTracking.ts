'use client';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { OnboardingSteps, steps, type OnboardingStep } from './types';
import useOnboardingStep from './useOnboardingStep';

export function useOnboardingStepCompletedTracking() {
  const trackingUser = useTrackingUser();
  if (!trackingUser) return null;

  const trackOnboardingStepCompleted = (
    step: OnboardingStep,
    metadata: Record<string, any> = {}
  ) => {
    trackEvent({
      name: 'app/onboarding.step.completed',
      data: {
        step,
        ...metadata,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  return { trackOnboardingStepCompleted };
}

export function useOnboardingTracking() {
  const trackingUser = useTrackingUser();
  const { lastCompletedStep } = useOnboardingStep();
  if (!trackingUser) return null;

  const trackOnboardingAction = (
    stepName?: OnboardingSteps,
    metadata: Record<string, any> = {}
  ) => {
    const step = steps.find((s) => s.name === stepName);

    trackEvent({
      name: 'app/onboarding.action',
      data: {
        step,
        lastCompletedStep: lastCompletedStep,
        ...metadata,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  return { trackOnboardingAction };
}
