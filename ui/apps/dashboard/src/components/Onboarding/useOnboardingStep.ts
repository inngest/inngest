import { useState } from 'react';
import { useLocalStorage } from 'react-use';

import { STEPS_ORDER, type OnboardingSteps } from './types';

export default function useOnboardingStep() {
  const [onboardingLastStepCompleted, setOnboardingLastStepCompleted] =
    useLocalStorage<OnboardingSteps>('onboardingLastStepCompleted', undefined);

  const [lastCompletedStep, setLastCompletedStep] = useState<OnboardingSteps | undefined>(
    onboardingLastStepCompleted
  );

  const isFinalStep = lastCompletedStep === STEPS_ORDER[STEPS_ORDER.length - 1];
  const nextStep = (
    !lastCompletedStep
      ? STEPS_ORDER[0]
      : isFinalStep
      ? lastCompletedStep
      : STEPS_ORDER[STEPS_ORDER.indexOf(lastCompletedStep) + 1]
  ) as OnboardingSteps;

  const updateLastCompletedStep = (step: OnboardingSteps) => {
    setOnboardingLastStepCompleted(step);
    setLastCompletedStep(step);
  };

  return { lastCompletedStep, updateLastCompletedStep, isFinalStep, nextStep };
}
