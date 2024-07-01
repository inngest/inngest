export function isNullish(value: unknown): value is null | undefined {
  return value === null || value === undefined;
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
