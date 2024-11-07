import { steps, type OnboardingSteps } from './types';

export function getNextStepName(currentStepName: OnboardingSteps): OnboardingSteps | undefined {
  const currentStep = steps.find((step) => step.name === currentStepName);

  if (!currentStep || currentStep.isFinalStep) {
    return undefined;
  }

  const nextStep = steps.find((step) => step.stepNumber === currentStep.stepNumber + 1);

  return nextStep ? nextStep.name : undefined;
}
