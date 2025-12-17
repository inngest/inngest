export const maxRenderedOutputSizeBytes = 1024 * 1024; // This prevents larger outputs from crashing the browser.

// Standard fields that appear on every Inngest event
export const STANDARD_EVENT_FIELDS = [
  'name',
  'data',
  'id',
  'ts',
  'ts_dt',
  'received_at',
  'received_at_dt',
  'v',
] as const;
