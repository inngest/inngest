'use client';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { OnboardingSteps, type TotalStepsCompleted } from './types';

export function useOnboardingTracking() {
  const trackingUser = useTrackingUser();
  if (!trackingUser) return null;

  const trackOnboardingStepCompleted = (
    step: OnboardingSteps,
    isFinalStep: boolean,
    completionSource?: string
  ) => {
    trackEvent({
      name: 'onboarding/step.completed',
      data: {
        step: step,
        isFinalStep: isFinalStep,
        completionSource: completionSource,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  const trackWidgetDismissed = (stepsCompleted: TotalStepsCompleted) => {
    trackEvent({
      name: 'onboarding/widget.dismissed',
      data: {
        stepsCompleted: stepsCompleted,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  const trackOnboardingOpened = (stepsCompleted: TotalStepsCompleted, source: string) => {
    trackEvent({
      name: 'onboarding/page.opened',
      data: {
        stepsCompleted: stepsCompleted,
        source: source,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  return { trackOnboardingStepCompleted, trackWidgetDismissed, trackOnboardingOpened };
}
