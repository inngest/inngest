import PrometheusSetupPage from '@/components/PrometheusIntegration/PrometheusSetupPage';
import { MetricsEntitlements } from '@/components/PrometheusIntegration/data';

export default async function Page() {
  const metricsEntitlements = await MetricsEntitlements();
  return (
    <PrometheusSetupPage
      metricsExportEnabled={metricsEntitlements.metricsExport.enabled}
      metricsGranularitySeconds={metricsEntitlements.metricsExportGranularity.limit}
    />
  );
}
