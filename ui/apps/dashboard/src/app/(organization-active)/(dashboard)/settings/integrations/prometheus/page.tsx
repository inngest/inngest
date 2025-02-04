import SetupPage from '@/components/PrometheusIntegration/SetupPage';
import { MetricsEntitlements } from '@/components/PrometheusIntegration/data';

export default async function Page() {
  const metricsEntitlements = await MetricsEntitlements();
  return (
    <SetupPage
      metricsExportEnabled={metricsEntitlements.metricsExport.enabled}
      metricsGranularitySeconds={metricsEntitlements.metricsExportGranularity.limit}
    />
  );
}
