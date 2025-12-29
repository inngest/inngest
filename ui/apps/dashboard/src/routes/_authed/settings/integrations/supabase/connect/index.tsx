import { createFileRoute, useNavigate } from '@tanstack/react-router';
import ConnectPage from '@inngest/components/PostgresIntegrations/ConnectPage';
import { connectContent } from '@inngest/components/PostgresIntegrations/Supabase/newSupabaseContent';
import { STEPS_ORDER } from '@inngest/components/PostgresIntegrations/newTypes';

export const Route = createFileRoute(
  '/_authed/settings/integrations/supabase/connect/',
)({
  component: SupabaseConnect,
});

function SupabaseConnect() {
  const navigate = useNavigate();
  const firstStep = STEPS_ORDER[0];

  return (
    <ConnectPage
      content={connectContent}
      onStartInstallation={() => {
        navigate({
          to: '/settings/integrations/supabase/$step',
          params: { step: firstStep },
        });
      }}
    />
  );
}
