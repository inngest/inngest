export function getFormattedJSONObjectOrArrayString(
  value: string,
): null | string {
  if (!mayBeJSONArray(value) && !mayBeJSONObject(value)) return null;

  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return null;
  }
}

function mayBeJSONArray(str: string): boolean {
  return str.startsWith("[") && str.endsWith("]");
}

function mayBeJSONObject(str: string): boolean {
  return str.startsWith("{") && str.endsWith("}");
}
