export function notNullish<T>(value: T): value is Exclude<T, null | undefined> {
  return value !== null && value !== undefined;
}
