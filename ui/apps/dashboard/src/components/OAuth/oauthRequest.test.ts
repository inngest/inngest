import { afterEach, expect, it, vi } from 'vitest';

import { oauthRequest } from './oauthRequest';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

it.each([
  ['<html>Bad gateway</html>', 502, 'Request failed.'],
  ['', 500, 'Request failed.'],
  ['null', 500, 'Request failed.'],
  [
    '{"error":"access_denied","error_description":"Sign in again."}',
    403,
    'Sign in again.',
  ],
  ['{"error":"access_denied"}', 403, 'access_denied'],
])('handles an error response: %s', async (body, status, message) => {
  vi.stubEnv('VITE_API_URL', 'https://api.inngest.test');
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(new Response(body, { status })),
  );
  await expect(
    oauthRequest(async () => null, '/oauth/sessions'),
  ).rejects.toThrow(message);
});

it('does not silently accept invalid success responses', async () => {
  vi.stubEnv('VITE_API_URL', 'https://api.inngest.test');
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('not json')));
  await expect(
    oauthRequest(async () => null, '/oauth/sessions'),
  ).rejects.toThrow(SyntaxError);
});
