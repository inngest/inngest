import { afterEach, describe, expect, it, vi } from 'vitest';

import { Route } from './chat';

const mocks = vi.hoisted(() => ({
  auth: vi.fn(),
  send: vi.fn(),
}));

vi.mock('@clerk/tanstack-react-start/server', () => ({
  auth: mocks.auth,
}));

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => (options: unknown) => ({ options }),
}));

vi.mock('@/lib/inngest/client', () => ({
  inngest: { send: mocks.send },
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

describe('POST /api/chat', () => {
  it('routes messages only to the authenticated user channel', async () => {
    mocks.auth.mockResolvedValue({ userId: 'user_authenticated' });
    mocks.send.mockResolvedValue(undefined);

    const response = await post({
      request: new Request('http://localhost/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userMessage: {
            id: '1841a625-4f86-4476-8f82-283fef830c42',
            content: 'show recent failures',
            role: 'user',
          },
          threadId: '2e297464-0211-4904-9f63-c9a494bf7b0f',
          userId: 'user_other',
          channelKey: 'insights:user_other',
        }),
      }),
    });

    expect(response.status).toBe(200);
    expect(mocks.send).toHaveBeenCalledWith({
      name: 'insights-agent/chat.requested',
      data: expect.objectContaining({
        userId: 'user_authenticated',
        channelKey: 'insights:user_authenticated',
      }),
      meta: {
        sessions: { thread_id: '2e297464-0211-4904-9f63-c9a494bf7b0f' },
      },
    });
  });

  it('rejects unauthenticated requests', async () => {
    mocks.auth.mockResolvedValue({ userId: null });

    const response = await post({
      request: new Request('http://localhost/api/chat', {
        method: 'POST',
      }),
    });

    expect(response.status).toBe(401);
    expect(mocks.send).not.toHaveBeenCalled();
  });
});
