const skipReasonLabels: Record<string, string> = {
  Singleton: 'Another run already in progress',
  FunctionPaused: 'Function is paused',
  FunctionDrained: 'Function is draining',
  FunctionBacklogSizeLimitHit: 'Backlog limit reached',
};

export function formatSkipReason(reason?: string): string {
  if (!reason) return 'Run was skipped';
  return skipReasonLabels[reason] ?? `Skipped: ${reason}`;
}
