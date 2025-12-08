import DatadogSetupPage from "@/app/(organization-active)/(dashboard)/settings/integrations/datadog/DatadogSetupPage";
import AddConnectionPage from "@/components/DatadogIntegration/AddConnectionPage";

export default async function Page() {
  return (
    <DatadogSetupPage
      subtitle={
        "Connect an environment to Datadog to send key metrics from Inngest to your Datadog account."
      }
      content={<AddConnectionPage />}
    />
  );
}
