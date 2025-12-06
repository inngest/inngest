import { entitlementSecondsToStr } from "@/utils/entitlementTimeFmt";

type Props = {
  metricsExportEnabled: boolean;
  granularitySeconds?: number;
  freshnessSeconds?: number;
};

export default function MetricsExportEntitlementValue({
  metricsExportEnabled,
  granularitySeconds,
  freshnessSeconds,
}: Props) {
  if (!metricsExportEnabled) {
    return "Not enabled";
  }

  return (
    <div className="flex items-center gap-2">
      <span className="font-medium">Enabled</span>
      {granularitySeconds && (
        <>
          <span className="border-subtle border-l border-r" />
          <span className="text-muted">Granularity</span>
          <span className="font-medium">
            {entitlementSecondsToStr(granularitySeconds)}
          </span>
        </>
      )}
      {freshnessSeconds && (
        <>
          <span className="border-subtle border-l border-r" />
          <span className="text-muted">Delay</span>
          <span className="font-medium">
            {entitlementSecondsToStr(freshnessSeconds)}
          </span>
        </>
      )}
    </div>
  );
}
