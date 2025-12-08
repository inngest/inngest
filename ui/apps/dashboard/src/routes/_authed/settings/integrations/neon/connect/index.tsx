import { createFileRoute, useNavigate } from '@tanstack/react-router';
import ConnectPage from '@inngest/components/PostgresIntegrations/NewConnectPage';
import { neonConnectContent } from '@inngest/components/PostgresIntegrations/Neon/newNeonContent';
import { STEPS_ORDER } from '@inngest/components/PostgresIntegrations/newTypes';

export const Route = createFileRoute(
  '/_authed/settings/integrations/neon/connect/',
)({
  component: NeonConnect,
});

function NeonConnect() {
  const navigate = useNavigate();
  const firstStep = STEPS_ORDER[0]!;

  return (
    <ConnectPage
      content={neonConnectContent}
      onStartInstallation={() => {
        navigate({
          to: '/settings/integrations/neon/$step',
          params: { step: firstStep },
        });
      }}
    />
  );
}
