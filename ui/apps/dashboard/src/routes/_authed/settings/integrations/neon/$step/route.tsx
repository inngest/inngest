import { createFileRoute, Outlet } from '@tanstack/react-router';

import { StepsProvider } from '@/components/PostgresIntegration/Context';
import PageHeader from '@/components/PostgresIntegration/PageHeader';
import StepsMenu from '@/components/PostgresIntegration/StepsMenu';

export const Route = createFileRoute(
  '/_authed/settings/integrations/neon/$step',
)({
  component: NeonStepLayout,
});

function NeonStepLayout() {
  const { step } = Route.useParams();

  return (
    <StepsProvider>
      <div className="text-subtle my-12 grid grid-cols-3">
        <main className="col-span-2 mx-20">
          <PageHeader step={step} integration="neon" />
          <Outlet />
        </main>
        <StepsMenu step={step} />
      </div>
    </StepsProvider>
  );
}
