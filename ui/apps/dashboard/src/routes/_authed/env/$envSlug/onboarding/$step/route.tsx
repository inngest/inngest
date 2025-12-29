import {
  createFileRoute,
  Outlet,
  redirect,
  useNavigate,
} from '@tanstack/react-router';

import PageHeader from '@/components/Onboarding/PageHeader';
import { isValidStep } from '@/components/Onboarding/types';
import { pathCreator } from '@/utils/urls';

import Menu from '@/components/Onboarding/Menu';

export const Route = createFileRoute('/_authed/env/$envSlug/onboarding/$step')({
  component: OnboardingStepLayout,
  loader: ({ params }) => {
    //
    // Onboarding is only available for production environment
    if (params.envSlug !== 'production') {
      redirect({
        to: '/env/$envSlug/apps',
        params: { envSlug: params.envSlug },
        throw: true,
      });
    }
  },
});

function OnboardingStepLayout() {
  const { envSlug, step } = Route.useParams();
  const navigate = useNavigate();

  if (!isValidStep(step)) {
    navigate({ to: pathCreator.onboarding() });
    return null;
  }

  return (
    <div className="text-basis my-12 grid grid-cols-3">
      <main className="col-span-2 mx-20">
        <PageHeader stepName={step} />
        <Outlet />
      </main>
      <Menu envSlug={envSlug} stepName={step} />
    </div>
  );
}
