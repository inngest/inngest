/**
 * Types for the Experiments UI components.
 */

export type ExperimentListItem = {
  experimentName: string;
  functionId: string;
  selectionStrategy: string;
  /**
   * Only populated by `useExperimentDetail` (detail page). The list query
   * does not fetch variants because the list UI doesn't display them.
   */
  variants?: string[];
  totalRuns: number;
  variantCount: number;
  firstSeen: Date;
  lastSeen: Date;
};
