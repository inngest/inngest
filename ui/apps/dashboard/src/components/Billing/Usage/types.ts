const usageDimensions = ['execution', 'run', 'step'] as const;
export type UsageDimension = (typeof usageDimensions)[number];
export function isUsageDimension(
  dimension: string,
): dimension is UsageDimension {
  return usageDimensions.includes(dimension as UsageDimension);
}
