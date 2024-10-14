'use client';

import { useRouter } from 'next/navigation';
import StepsPageHeader from '@inngest/components/Steps/StepsPageHeader';

import { onboardingMenuStepContent } from '@/components/Onboarding/content';
import { STEPS_ORDER, isValidStep } from '@/components/Onboarding/types';
import { pathCreator } from '@/utils/urls';

export default function PageHeader({ step }: { step: string }) {
  const router = useRouter();
  if (!isValidStep(step)) {
    router.push(pathCreator.neonIntegrationStep({}));
    return;
  }
  const currentStep = STEPS_ORDER.indexOf(step);
  const stepContent = onboardingMenuStepContent.step[step];

  return (
    <StepsPageHeader
      currentStep={currentStep + 1}
      totalSteps={STEPS_ORDER.length}
      title={stepContent.title}
    />
  );
}
