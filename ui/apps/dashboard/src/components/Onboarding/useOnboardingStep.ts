import { useState } from 'react';
import { useLocalStorage } from 'react-use';

import {
  type OnboardingSteps,
  type OnboardingStepsCompleted,
  type OnboardingStepsString,
} from './types';

export default function useOnboardingStep() {
  const [onboardingLastStepCompleted, setOnboardingLastStepCompleted] =
    useLocalStorage<OnboardingStepsString>('onboardingLastStepCompleted', undefined);

  const [lastCompletedStep, setLastCompletedStep] = useState<OnboardingStepsCompleted>(
    onboardingLastStepCompleted ? (Number(onboardingLastStepCompleted) as OnboardingSteps) : 0
  );

  const isFinalStep = lastCompletedStep === 4;
  const nextStep = isFinalStep ? lastCompletedStep : lastCompletedStep + 1;

  const updateLastCompletedStep = (step: OnboardingSteps) => {
    const stepString: OnboardingStepsString = step.toString() as OnboardingStepsString;
    setOnboardingLastStepCompleted(stepString);
    setLastCompletedStep(step);
  };

  return { lastCompletedStep, updateLastCompletedStep, isFinalStep, nextStep };
}
