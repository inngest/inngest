import { Header } from '@inngest/components/Header/NewHeader';
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router';

import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/onboarding')({
  component: OnboardingLayout,
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
