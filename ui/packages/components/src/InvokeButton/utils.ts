function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function parseCode(code: string): {
  data: Record<string, unknown>;
  user: Record<string, unknown> | null;
  meta: { sessions: Record<string, string> } | null;
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

  let meta: { sessions: Record<string, string> } | null = null;
  if (payload.meta != null) {
    if (!isRecord(payload.meta)) {
      throw new Error('The "meta" field must be an object or null');
    }

    const supportedMetaKeys = ['sessions'];
    for (const key of Object.keys(payload.meta)) {
      if (!supportedMetaKeys.includes(key)) {
        throw new Error(`Property "meta.${key}" is not supported when invoking a function`);
      }
    }

    if (payload.meta.sessions != null) {
      if (!isRecord(payload.meta.sessions)) {
        throw new Error('The "meta.sessions" field must be an object or null');
      }

      const sessions: Record<string, string> = {};
      for (const [key, value] of Object.entries(payload.meta.sessions)) {
        if (typeof value !== 'string' && typeof value !== 'number') {
          throw new Error(
            'The "meta.sessions" field must be an object with string or number values'
          );
        }
        if (typeof value === 'number' && !Number.isFinite(value)) {
          throw new Error(`The "meta.sessions.${key}" field must be a finite number`);
        }

        sessions[key] = String(value);
      }

      meta = { sessions };
    }
  }

  const supportedKeys = ['data', 'user', 'meta'];
  for (const key of Object.keys(payload)) {
    if (!supportedKeys.includes(key)) {
      throw new Error(`Property "${key}" is not supported when invoking a function`);
    }
  }

  return { data, user, meta };
}
