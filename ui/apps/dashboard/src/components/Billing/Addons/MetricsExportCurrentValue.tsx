type Props = {
  metricsExportEnabled: boolean;
  granularitySeconds?: number;
  freshnessSeconds?: number;
};

export default function MetricsExportValue({
  metricsExportEnabled,
  granularitySeconds,
  freshnessSeconds,
}: Props) {
  if (!metricsExportEnabled) {
    return 'Not enabled';
  }

  if (!granularitySeconds || !freshnessSeconds) {
    throw new Error(
      'granularitySeconds and freshnessSeconds must be given when metrics export is enabled'
    );
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
        <span
          className="border-subtle"
          style={{ borderLeftWidth: '1px', borderRightWidth: '1px' }}
        ></span>
        <span className="text-muted">Granularity</span>
        <span className="font-medium">{secondsToStr(granularitySeconds)}</span>
        <span
          className="border-subtle"
          style={{ borderLeftWidth: '1px', borderRightWidth: '1px' }}
        ></span>
        <span className="text-muted">Freshness</span>
        <span className="font-medium">{secondsToStr(freshnessSeconds)}</span>
      </div>
    </>
  );
}

function secondsToStr(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m == 0) {
    return `${s} sec`;
  }
  if (s == 0) {
    return `${m} mins`;
  }
  return `${m} mins ${s} sec`;
}
