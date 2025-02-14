import SetupPage from '@/components/DatadogIntegration/SetupPage';
import { MetricsEntitlements } from '@/components/PrometheusIntegration/data';

export default async function Page() {
  const metricsEntitlements = await MetricsEntitlements();
  return <SetupPage metricsExportEnabled={metricsEntitlements.metricsExport.enabled} />;
}
