/**
 * Returns the values of an enum in the order you pass them in while making sure all values are
 * present. If an enum value is missing, you will get a TypeScript error.
 *
 * This is useful for when you need an exhaustive list of enum values in a specific order.
 *
 * @param _enumType - The enum type to get the values from
 * @param enumValues - The enum values in the order you want them returned
 * @returns The enum values in the order you passed them in
 */
export default function getOrderedEnumValues<U extends string, T extends { [K in keyof T]: U }>(
  _enumType: T,
  enumValues: [...U[]]
): U[] {
  return enumValues;
}
