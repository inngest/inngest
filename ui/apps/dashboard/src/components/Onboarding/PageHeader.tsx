'use client';

import { useRouter } from 'next/navigation';
import StepsPageHeader from '@inngest/components/Steps/StepsPageHeader';

import { onboardingMenuStepContent } from '@/components/Onboarding/content';
import { isValidStep, steps } from '@/components/Onboarding/types';
import { pathCreator } from '@/utils/urls';

export default function PageHeader({ stepName }: { stepName: string }) {
  const router = useRouter();
  if (!isValidStep(stepName)) {
    router.push(pathCreator.onboarding());
    return;
  }
  const currentStep = steps.find((step) => step.name === stepName)?.stepNumber;
  const stepContent = onboardingMenuStepContent.step[stepName];

  return (
    <StepsPageHeader
      currentStep={currentStep || 1}
      totalSteps={steps.length}
      title={stepContent.title}
    />
  );
}
