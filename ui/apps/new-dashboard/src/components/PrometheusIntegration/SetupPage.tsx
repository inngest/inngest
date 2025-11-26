import { Link } from "@inngest/components/Link/NewLink";
import { IconPrometheus } from "@inngest/components/icons/platforms/Prometheus";

import IntegrationNotEnabledMessage from "@/components/Integration/IntegrationNotEnabledMessage";
import MetricsExportEntitlementBanner from "@/components/Integration/MetricsExportEntitlementsBanner";
import ConfigSteps from "@/components/PrometheusIntegration/ConfigSteps";

type Props = {
  metricsExportEnabled: boolean;
  metricsGranularitySeconds: number;
  metricsFreshnessSeconds: number;
};

export default function SetupPage({
  metricsExportEnabled,
  metricsGranularitySeconds,
  metricsFreshnessSeconds,
}: Props) {
  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="text-basis mb-7 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
          <IconPrometheus className="text-onContrast" size={20} />
        </div>
        Prometheus
      </div>

      <div className="text-muted mb-6 w-full text-base font-normal">
        This integration allows your Prometheus server to scrape metrics about
        your Inngest environment.{" "}
        <Link
          target="_blank"
          size="medium"
          href="https://www.inngest.com/docs/platform/monitor/prometheus-metrics-export-integration"
        >
          Learn more
        </Link>
      </div>

      {metricsExportEnabled ? (
        <>
          <MetricsExportEntitlementBanner
            granularitySeconds={metricsGranularitySeconds}
            freshnessSeconds={metricsFreshnessSeconds}
            className={"mb-6"}
          />
          <ConfigSteps metricsGranularitySeconds={metricsGranularitySeconds} />
        </>
      ) : (
        <IntegrationNotEnabledMessage integrationName="Prometheus" />
      )}
    </div>
  );
}
