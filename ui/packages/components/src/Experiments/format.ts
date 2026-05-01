// Variant weights are floats: an even three-way split arrives as 33.3, and
// raw float arithmetic from the SDK can surface drift like 0.30000000000000004.
// Render integers as-is and trim everything else to one decimal place.
export const formatVariantWeight = (weight: number): string =>
  Number.isInteger(weight) ? String(weight) : weight.toFixed(1);
