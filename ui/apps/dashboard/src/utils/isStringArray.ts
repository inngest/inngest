export default function isStringArray(value: unknown): value is Array<string> {
  if (!Array.isArray(value)) {
    return false;
  }

  return value.every((item) => typeof item === 'string');
}
