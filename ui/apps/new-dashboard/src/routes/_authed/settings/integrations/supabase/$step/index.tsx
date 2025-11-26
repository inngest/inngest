import { createFileRoute, useNavigate } from "@tanstack/react-router";
import Auth from "@inngest/components/PostgresIntegrations/Neon/NewAuth";
import Connect from "@inngest/components/PostgresIntegrations/Neon/NewConnect";
import {
  IntegrationSteps,
  STEPS_ORDER,
} from "@inngest/components/PostgresIntegrations/newTypes";
import { toast } from "sonner";

import { useSteps } from "@/components/PostgresIntegration/Context";
import {
  testAuth,
  testAutoSetup,
  verifyAutoSetup,
  verifyCredentials,
} from "@/queries/server-only/integrations/db";

export const Route = createFileRoute(
  "/_authed/settings/integrations/supabase/$step/",
)({
  component: SupabaseStep,
});

function SupabaseStep() {
  const { step } = Route.useParams();
  const { setStepsCompleted, credentials, setCredentials } = useSteps();
  const navigate = useNavigate();
  const firstStep = STEPS_ORDER[0]!;

  function handleLostCredentials() {
    toast.error("Lost credentials. Going back to the first step.");
    navigate({
      to: "/settings/integrations/supabase/$step",
      params: { step: firstStep },
    });
  }

  if (step === IntegrationSteps.Authorize) {
    return (
      <Auth
        savedCredentials={credentials}
        integration="supabase"
        onSuccess={(value) => {
          setCredentials(value);
          setStepsCompleted(IntegrationSteps.Authorize);
        }}
        verifyCredentials={(input) => verifyCredentials({ data: { input } })}
        nextStep={IntegrationSteps.ConnectDb}
      />
    );
  } else if (step === IntegrationSteps.ConnectDb) {
    return (
      <Connect
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.ConnectDb);
        }}
        integration="supabase"
        // @ts-expect-error - TANSTACK TODO: sort out type issue
        verifyAutoSetup={(input) => verifyAutoSetup({ data: { input } })}
        savedCredentials={credentials}
        handleLostCredentials={handleLostCredentials}
      />
    );
  }

  return <div>Page Content</div>;
}
