import DatadogSetupPage from "@/app/(organization-active)/(dashboard)/settings/integrations/datadog/DatadogSetupPage";
import StartPage from "@/components/DatadogIntegration/StartPage";

export default async function Page() {
  return (
    <DatadogSetupPage
      title={"Connect to Datadog"}
      showSubtitleDocsLink={false}
      content={<StartPage />}
    />
  );
}
