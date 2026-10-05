export async function oauthRequest<T = unknown>(
  getToken: () => Promise<string | null>,
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const token = await getToken();
  signal?.throwIfAborted();
  const response = await fetch(new URL(path, import.meta.env.VITE_API_URL), {
    method: body === undefined ? 'GET' : 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  });
  const payload = (await response.json()) as T & {
    error?: string;
    error_description?: string;
  };
  if (!response.ok) {
    throw new Error(
      payload.error_description ?? payload.error ?? 'Request failed.',
    );
  }
  return payload;
}
