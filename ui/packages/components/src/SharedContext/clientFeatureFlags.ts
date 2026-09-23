// Register browser-consumed flags here. Before merging a new entry, enable
// "SDKs using client-side ID" for the flag in LaunchDarkly.
export const clientFeatureFlags = [
  'advanced-observability',
  'ai-overview-dashboard',
  'connect-worker-concurrency-metrics',
  'dedicated-slack-channel',
  'enable-step-metadata',
  'incident-banner',
  'insights-charts',
  'legacy-scores-page-enabled',
  'overdue-invoice-banner',
  'polling-disabled',
  'rest-runs-table',
  'supabase-integration',
  'traces-preview',
] as const;

export type ClientFeatureFlagKey = (typeof clientFeatureFlags)[number];
