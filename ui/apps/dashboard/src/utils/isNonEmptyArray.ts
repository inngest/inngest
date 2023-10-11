export type NonEmptyArray<T> = [...Array<T>];

export default function isNonEmptyArray<T extends unknown>(
  array: Array<T>
): array is NonEmptyArray<T> {
  return array.length > 0;
}
