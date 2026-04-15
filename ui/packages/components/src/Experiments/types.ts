/**
 * Types for the Experiments UI components.
 */

export type ExperimentListItem = {
  experimentName: string;
  functionId: string;
  selectionStrategy: string;
  variants: string[];
  totalRuns: number;
  variantCount: number;
  firstSeen: Date;
  lastSeen: Date;
};
