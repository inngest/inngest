import { useState } from 'react';
import { useLocalStorage } from 'react-use';

import { type OnboardingSteps, type OnboardingStepsString } from './types';

export default function useOnboardingStep() {
  const [onboardingLastStepCompleted, setOnboardingLastStepCompleted] =
    useLocalStorage<OnboardingStepsString>('onboardingLastStepCompleted', undefined);

  const [lastCompletedStep, setLastCompletedStep] = useState<OnboardingSteps | 0>(
    onboardingLastStepCompleted ? (Number(onboardingLastStepCompleted) as OnboardingSteps) : 0
  );

  const updateLastCompletedStep = (step: OnboardingSteps) => {
    const stepString: OnboardingStepsString = step.toString() as OnboardingStepsString;
    setOnboardingLastStepCompleted(stepString);
    setLastCompletedStep(step);
  };

  return { lastCompletedStep, updateLastCompletedStep };
}
