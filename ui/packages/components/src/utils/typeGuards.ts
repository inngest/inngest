function isObject(value: unknown): value is object {
  return typeof value === 'object' && !Array.isArray(value) && value !== null;
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  if (!isObject(value)) {
    return false;
  }

  for (const key in value) {
    if (typeof key !== 'string') {
      return false;
    }
  }

  return true;
}
