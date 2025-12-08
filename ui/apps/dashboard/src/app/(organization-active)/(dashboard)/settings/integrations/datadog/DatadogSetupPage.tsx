import type { ReactNode } from "react";
import { IconDatadog } from "@inngest/components/icons/platforms/Datadog";

import IntegrationNotEnabledMessage from "@/components/Integration/IntegrationNotEnabledMessage";
import MetricsExportEntitlementBanner from "@/components/Integration/MetricsExportEntitlementsBanner";
import { MetricsEntitlements } from "@/components/PrometheusIntegration/data";

type Props = {
  title?: string;
  subtitle?: string;
  showSubtitleDocsLink?: boolean;
  showEntitlements?: boolean;
  content: ReactNode;
};

export default async function DatadogSetupPage({
  title = "Datadog",
  subtitle,
  showSubtitleDocsLink = true,
  showEntitlements = false,
  content,
}: Props) {
  const metricsEntitlements = await MetricsEntitlements();
  const featureAvailable = metricsEntitlements.metricsExport.enabled;

  if (showSubtitleDocsLink && !subtitle) {
    throw new Error(
      "programming error: without a subtitle, docs link will not be shown",
    );
  }

  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="text-basis mb-7 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
          <IconDatadog className="text-onContrast" size={20} />
        </div>
        {title}
      </div>

      {subtitle && (
        <div className="text-muted mb-6 w-full text-base font-normal">
          {subtitle}
          {/* TODO(cdzombak): Link to Datadog docs, once we've written them */}
          {/*{showSubtitleDocsLink && (*/}
          {/*  <Link target="_blank" size="medium" href="https://www.inngest.com/docs/">*/}
          {/*    Read documentation*/}
          {/*  </Link>*/}
          {/*)}*/}
        </div>
      )}

      <div className="text-sm font-normal">
        {!featureAvailable && (
          <IntegrationNotEnabledMessage integrationName="Datadog" />
        )}

        {featureAvailable && showEntitlements && (
          <MetricsExportEntitlementBanner
            granularitySeconds={
              metricsEntitlements.metricsExportGranularity.limit
            }
            freshnessSeconds={metricsEntitlements.metricsExportFreshness.limit}
            className={"mb-12"}
          />
        )}

        {featureAvailable && content}
      </div>
    </div>
  );
}
