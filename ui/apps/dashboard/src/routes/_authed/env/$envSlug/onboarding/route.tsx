import { Header } from '@inngest/components/Header/NewHeader';
import { createFileRoute, Outlet, redirect } from '@tanstack/react-router';

import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/onboarding')({
  component: OnboardingLayout,
  loader: ({ params }) => {
    //
    // Always redirect to production environment for onboarding
    redirect({
      to: '/env/$envSlug/onboarding/$step',
      params: { envSlug: 'production', step: 'create-app' },
      throw: true,
    });
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
