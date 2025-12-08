import DatadogSetupPage from "@/app/(organization-active)/(dashboard)/settings/integrations/datadog/DatadogSetupPage";
import FinishPage from "@/components/DatadogIntegration/FinishPage";

export default async function Page() {
  return (
    <DatadogSetupPage
      title={"Connect to Datadog"}
      showSubtitleDocsLink={false}
      content={<FinishPage />}
    />
  );
}
