import { afterEach, describe, expect, it, vi } from 'vitest';

import { Route } from './token';

const mocks = vi.hoisted(() => ({
  auth: vi.fn(),
  getClientSubscriptionToken: vi.fn(),
}));

vi.mock('@clerk/tanstack-react-start/server', () => ({
  auth: mocks.auth,
}));

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => (options: unknown) => ({ options }),
}));

vi.mock('inngest/react', () => ({
  getClientSubscriptionToken: mocks.getClientSubscriptionToken,
}));

vi.mock('@/lib/inngest/client', () => ({
  inngest: {},
}));

type PostHandler = (args: { request: Request }) => Promise<Response>;

const post = (
  Route.options as unknown as {
    server: { handlers: { POST: PostHandler } };
  }
).server.handlers.POST;

afterEach(() => {
  vi.clearAllMocks();
});

describe('POST /api/realtime/token', () => {
  it('signs only the authenticated user channel', async () => {
    mocks.auth.mockResolvedValue({ userId: 'user_authenticated' });
    mocks.getClientSubscriptionToken.mockResolvedValue({ token: 'token' });

    const response = await post({
      request: new Request('http://localhost/api/realtime/token', {
        method: 'POST',
        body: JSON.stringify({ channelKey: 'insights:user_other' }),
      }),
    });

    expect(response.status).toBe(200);
    expect(mocks.getClientSubscriptionToken).toHaveBeenCalledWith(
      expect.anything(),
      {
        channel: expect.objectContaining({
          name: 'insights:user_authenticated',
        }),
        topics: ['agent_stream'],
      },
    );
  });

  it('rejects unauthenticated requests', async () => {
    mocks.auth.mockResolvedValue({ userId: null });

    const response = await post({
      request: new Request('http://localhost/api/realtime/token', {
        method: 'POST',
      }),
    });

    expect(response.status).toBe(401);
    expect(mocks.getClientSubscriptionToken).not.toHaveBeenCalled();
  });
});
