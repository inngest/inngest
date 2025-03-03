import { entitlementSecondsToStr } from '@/utils/entitlementTimeFmt';

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
    return 'Not enabled';
  }

  return (
    <>
      <div
        style={{
          display: 'flex',
          flexDirection: 'row',
          gap: '0.5rem',
        }}
      >
        <span className="font-medium">Enabled</span>

        {granularitySeconds && (
          <span>
            <span
              className="border-subtle"
              style={{ borderLeftWidth: '1px', borderRightWidth: '1px' }}
            ></span>
            <span className="text-muted">Granularity</span>
            <span className="font-medium">{entitlementSecondsToStr(granularitySeconds)}</span>
          </span>
        )}
        {freshnessSeconds && (
          <span>
            <span
              className="border-subtle"
              style={{ borderLeftWidth: '1px', borderRightWidth: '1px' }}
            ></span>
            <span className="text-muted">Delay</span>
            <span className="font-medium">{entitlementSecondsToStr(freshnessSeconds)}</span>
          </span>
        )}
      </div>
    </>
  );
}
