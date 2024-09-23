'use client';

import { useRouter } from 'next/navigation';
import { neonMenuStepContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';
import { STEPS_ORDER, isValidStep } from '@inngest/components/PostgresIntegrations/types';
import StepsPageHeader from '@inngest/components/Steps/StepsPageHeader';

import { pathCreator } from '@/utils/urls';

export default function PageHeader({ step }: { step: string }) {
  const router = useRouter();
  if (!isValidStep(step)) {
    router.push(pathCreator.neonIntegrationStep({}));
    return;
  }
  const currentStep = STEPS_ORDER.indexOf(step);
  const stepContent = neonMenuStepContent.step[step];

  return (
    <StepsPageHeader
      currentStep={currentStep + 1}
      totalSteps={STEPS_ORDER.length}
      title={stepContent.title}
    />
  );
}
