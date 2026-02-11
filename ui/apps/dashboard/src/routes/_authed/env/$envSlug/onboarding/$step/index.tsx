import { createFileRoute } from '@tanstack/react-router';

import CreateApp from '@/components/Onboarding/CreateApp';
import DeployApp from '@/components/Onboarding/DeployApp';
import SyncApp from '@/components/Onboarding/SyncApp';
import { OnboardingSteps } from '@/components/Onboarding/types';
import InvokeFn from '@/components/Onboarding/InvokeFn';

type OnboardingStepSearchParams = {
  nonVercel?: string;
};

export const Route = createFileRoute('/_authed/env/$envSlug/onboarding/$step/')(
  {
    component: OnboardingStepPage,
    validateSearch: (
      search: Record<string, unknown>,
    ): OnboardingStepSearchParams => ({
      nonVercel: search.nonVercel as string | undefined,
    }),
  },
);

function OnboardingStepPage() {
  const { step } = Route.useParams();

  if (step === OnboardingSteps.CreateApp) {
    return <CreateApp />;
  } else if (step === OnboardingSteps.DeployApp) {
    return <DeployApp />;
  } else if (step === OnboardingSteps.SyncApp) {
    return <SyncApp />;
  } else if (step === OnboardingSteps.InvokeFn) {
    return <InvokeFn />;
  }

  return <div>Page Content</div>;
}
