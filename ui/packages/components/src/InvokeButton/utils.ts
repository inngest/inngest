function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function parseCode(code: string): {
  data: Record<string, unknown>;
  user: Record<string, unknown> | null;
} {
  if (typeof code !== 'string') {
    throw new Error("The payload form field isn't a string");
  }

  let payload: Record<string, unknown>;
  const parsed: unknown = JSON.parse(code);
  if (!isRecord(parsed)) {
    throw new Error('Parsed JSON is not an object');
  }

  payload = parsed;

  let { data } = payload;
  if (data === null) {
    data = {};
  }
  if (!isRecord(data)) {
    throw new Error('The "data" field must be an object or null');
  }

  let user: Record<string, unknown> | null = null;
  if (payload.user) {
    if (!isRecord(payload.user)) {
      throw new Error('The "user" field must be an object or null');
    }
    user = payload.user;
  }

  const supportedKeys = ['data', 'user'];
  for (const key of Object.keys(payload)) {
    if (!supportedKeys.includes(key)) {
      throw new Error(`Property "${key}" is not supported when invoking a function`);
    }
  }

  return { data, user };
}
