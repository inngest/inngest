import { createFileRoute, Outlet } from '@tanstack/react-router';

import { StepsProvider } from '@/components/PostgresIntegration/Context';
import PageHeader from '@/components/PostgresIntegration/PageHeader';
import StepsMenu from '@/components/PostgresIntegration/StepsMenu';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';

// SUpabase has two steps.
const steps = [IntegrationSteps.Authorize, IntegrationSteps.ConnectDb];

export const Route = createFileRoute(
  '/_authed/settings/integrations/supabase/$step',
)({
  component: SupabaseStepLayout,
});

function SupabaseStepLayout() {
  const { step } = Route.useParams();

  return (
    <StepsProvider>
      <div className="text-subtle my-12 grid grid-cols-3">
        <main className="col-span-2 mx-20">
          <PageHeader step={step} steps={steps} integration="supabase" />
          <Outlet />
        </main>
        <StepsMenu step={step} steps={steps} />
      </div>
    </StepsProvider>
  );
}
