export type Features = {
  history: number;

  /**
   * Whether to use the traces Developer Preview to view traces instead of the
   * current view.
   */
  tracesPreview?: boolean;

  /**
   * Whether to use the V4 run details UI (composable timeline bar).
   */
  runDetailsV4?: boolean;
};
