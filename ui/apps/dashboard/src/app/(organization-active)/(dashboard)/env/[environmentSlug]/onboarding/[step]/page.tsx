'use client';

import CreateApp from '@/components/Onboarding/CreateApp';
import DeployApp from '@/components/Onboarding/DeployApp';
import InvokeFn from '@/components/Onboarding/InvokeFn';
import { OnboardingSteps } from '@/components/Onboarding/types';

export default function OnboardingStep({ params: { step } }: { params: { step: string } }) {
  if (step === OnboardingSteps.CreateApp) {
    return <CreateApp />;
  } else if (step === OnboardingSteps.DeployApp) {
    return <DeployApp />;
  } else if (step === OnboardingSteps.SyncApp) {
    return <p>Sync component</p>;
  } else if (step === OnboardingSteps.InvokeFn) {
    return <InvokeFn />;
  }

  return <div>Page Content</div>;
}