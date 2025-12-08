import DatadogSetupPage from "@/app/(organization-active)/(dashboard)/settings/integrations/datadog/DatadogSetupPage";
import SetupPage from "@/components/DatadogIntegration/SetupPage";

export default async function Page() {
  return (
    <DatadogSetupPage
      subtitle={"Send key Inngest metrics directly to your Datadog account."}
      showEntitlements={true}
      content={<SetupPage />}
    />
  );
}
