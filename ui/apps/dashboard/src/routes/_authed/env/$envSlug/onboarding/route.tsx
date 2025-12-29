import { Header } from '@inngest/components/Header/NewHeader';
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router';

import { pathCreator } from '@/utils/urls';
import {
  OnboardingSteps,
  type OnboardingStep,
} from '@/components/Onboarding/types';

export const Route = createFileRoute('/_authed/env/$envSlug/onboarding')({
  component: OnboardingLayout,
  loader: ({
    params,
  }: {
    params: { step?: OnboardingStep; envSlug: string };
  }) => {
    if (!params.step) {
      redirect({
        to: pathCreator.onboardingSteps({
          envSlug: params.envSlug,
          step: OnboardingSteps.CreateApp,
        }),
        throw: true,
      });
    }
  },
});

function OnboardingLayout() {
  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Getting started', href: pathCreator.onboarding() },
        ]}
      />
      <Outlet />
    </>
  );
}
