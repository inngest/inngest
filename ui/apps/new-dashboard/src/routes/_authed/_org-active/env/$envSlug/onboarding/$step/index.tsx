import { createFileRoute } from "@tanstack/react-router";

import CreateApp from "@/components/Onboarding/CreateApp";
import DeployApp from "@/components/Onboarding/DeployApp";
import SyncApp from "@/components/Onboarding/SyncApp";
import { OnboardingSteps } from "@/components/Onboarding/types";
import InvokeFn from "@/components/Onboarding/InvokeFn";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/onboarding/$step/",
)({
  component: OnboardingStepPage,
});

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
