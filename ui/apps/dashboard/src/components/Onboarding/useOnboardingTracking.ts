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

  const trackCreateAppAction = (type: string) => {
    trackEvent({
      name: 'onboarding/create-app.action',
      data: {
        type: type,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  const trackDeployAppAction = (type: string, hostingProvider: string) => {
    trackEvent({
      name: 'onboarding/deploy-app.action',
      data: {
        type: type,
        hostingProvider: hostingProvider,
      },
      user: trackingUser,
      v: '2024-11-04.1',
    });
  };

  return {
    trackOnboardingStepCompleted,
    trackWidgetDismissed,
    trackOnboardingOpened,
    trackCreateAppAction,
    trackDeployAppAction,
  };
}
