import NeonAuth from "@inngest/components/PostgresIntegrations/Neon/NewAuth";
import NeonConnect from "@inngest/components/PostgresIntegrations/Neon/NewConnect";
import NeonFormat from "@inngest/components/PostgresIntegrations/Neon/NewFormat";
import {
  IntegrationSteps,
  STEPS_ORDER,
} from "@inngest/components/PostgresIntegrations/newTypes";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useSteps } from "@/components/PostgresIntegration/Context";
import {
  verifyAutoSetup,
  verifyCredentials,
  verifyLogicalReplication,
} from "@/queries/server/integrations/db";

export const Route = createFileRoute(
  "/_authed/settings/integrations/neon/$step/",
)({
  component: NeonStep,
});

function NeonStep() {
  const { step } = Route.useParams();
  const { setStepsCompleted, credentials, setCredentials } = useSteps();
  const navigate = useNavigate();
  const firstStep = STEPS_ORDER[0]!;

  function handleLostCredentials() {
    toast.error("Lost credentials. Going back to the first step.");
    navigate({
      to: "/settings/integrations/neon/$step",
      params: { step: firstStep },
    });
  }

  if (step === IntegrationSteps.Authorize) {
    return (
      <NeonAuth
        savedCredentials={credentials}
        onSuccess={(value) => {
          setCredentials(value);
          setStepsCompleted(IntegrationSteps.Authorize);
        }}
        integration="neon"
        verifyCredentials={(input) => verifyCredentials({ data: { input } })}
      />
    );
  } else if (step === IntegrationSteps.FormatWal) {
    return (
      <NeonFormat
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.FormatWal);
        }}
        integration="neon"
        verifyLogicalReplication={(input) =>
          verifyLogicalReplication({ data: { input } })
        }
        savedCredentials={credentials}
        handleLostCredentials={handleLostCredentials}
      />
    );
  } else if (step === IntegrationSteps.ConnectDb) {
    return (
      <NeonConnect
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.ConnectDb);
        }}
        integration="neon"
        verifyAutoSetup={(input) => verifyAutoSetup({ data: { input } })}
        savedCredentials={credentials}
        handleLostCredentials={handleLostCredentials}
      />
    );
  }

  return <div>Page Content</div>;
}
