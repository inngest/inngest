'use client';

import { OnboardingSteps } from '@/components/Onboarding/types';

export default function OnboardingStep({ params: { step } }: { params: { step: string } }) {
  if (step === OnboardingSteps.CreateApp) {
    return <p>Create component</p>;
  } else if (step === OnboardingSteps.DeployApp) {
    return <p>Deploy component</p>;
  } else if (step === OnboardingSteps.SyncApp) {
    return <p>Sync component</p>;
  } else if (step === OnboardingSteps.InvokeFn) {
    return <p>Invoke component</p>;
  }

  return <div>Page Content</div>;
}
