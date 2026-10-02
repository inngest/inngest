// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ImagesPanel } from './ImagesPanel';

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  environment: { id: 'workspace', slug: 'staging' },
}));
vi.mock('@/queries/useInngestAPIFetch', () => ({
  useInngestAPIFetch: () => mocks.fetch,
}));
vi.mock('@/components/Environments/environment-context', () => ({
  useEnvironment: () => mocks.environment,
}));

const id = '22222222-2222-4222-8222-222222222222';
const sha = 'a'.repeat(64);
const tag = {
  name: 'latest',
  digest: sha,
  generation: '7',
  architecture: 'amd64',
  immutable: false,
  deleted: false,
};
const image = {
  id,
  name: 'app',
  scope: 'workspace',
  createdAt: '2026-10-01T00:00:00Z',
  tags: [tag],
  artifacts: [
    {
      digest: sha,
      state: 'ready',
      createdAt: '2026-10-01T00:00:00Z',
      manifest: {
        architecture: 'amd64',
        rootfsSha256: sha,
        sizeBytes: '2097152',
        config: { cmd: ['/app'] },
      },
    },
  ],
};
const build = {
  id,
  imageId: id,
  name: 'app',
  tag: 'latest',
  status: 'pending',
  state: 'queued',
  architecture: 'amd64',
  sourceType: 'docker_export',
  uploadSizeBytes: '1024',
  createdAt: image.createdAt,
  logs: '<script>privateCode()</script>',
};
const json = (body: unknown) =>
  new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
  });
let client: QueryClient;
let rejectDelete = false;

beforeEach(() => {
  vi.clearAllMocks();
  rejectDelete = false;
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  mocks.fetch.mockImplementation(async (path: string, init?: RequestInit) => {
    if (init?.method === 'DELETE')
      return rejectDelete
        ? new Response('private provider error', { status: 409 })
        : json({
            data: { ...tag, generation: '8', digest: null, deleted: true },
          });
    if (path.endsWith('/cancel'))
      return json({ data: { ...build, status: 'failed', state: 'cancelled' } });
    if (path.startsWith('/v2/images?')) return json({ data: [image] });
    if (path.startsWith('/v2/images/app?')) return json({ data: image });
    if (path.startsWith('/v2/image-builds?')) return json({ data: [build] });
    if (path.startsWith('/v2/image-builds/')) return json({ data: build });
    if (path === '/v2/image-usage')
      return json({
        data: {
          storedBytes: '2097152',
          uploadBytes: '1024',
          buildMilliseconds: '60000',
          activeBuilds: 1,
        },
      });
    throw new Error(`Unexpected request ${path}`);
  });
});
afterEach(() => {
  cleanup();
  client.clear();
});
const show = () =>
  render(
    <QueryClientProvider client={client}>
      <ImagesPanel />
    </QueryClientProvider>,
  );

describe('image dashboard', () => {
  it('shows private images, storage, pre-upload builds and escaped logs', async () => {
    const { container } = show();
    await screen.findByText('This environment');
    expect(screen.getByText('Awaiting upload')).toBeTruthy();
    expect(screen.getByText('2.0 MiB')).toBeTruthy();
    expect(screen.getByText('1.00')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'app:latest' }));
    await screen.findByLabelText('Build logs');
    expect(screen.getByLabelText('Build logs').textContent).toContain(
      '<script>',
    );
    expect(container.querySelector('script')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel build' }));
    await waitFor(() =>
      expect(mocks.fetch).toHaveBeenCalledWith(
        `/v2/image-builds/${id}/cancel`,
        expect.objectContaining({ method: 'POST' }),
      ),
    );
  });

  it('confirms deletion and preserves the observed generation when a concurrent edit wins', async () => {
    rejectDelete = true;
    show();
    fireEvent.click(
      await screen.findByRole('button', { name: 'app' }),
    );
    fireEvent.click(
      await screen.findByRole('button', { name: 'Delete tag latest' }),
    );
    expect(
      mocks.fetch.mock.calls.some(([, init]) => init?.method === 'DELETE'),
    ).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: 'Confirm delete tag' }));
    await screen.findByRole('alert');
    expect(screen.getByRole('alert').textContent).toContain(
      'This image changed',
    );
    expect(screen.getByRole('alert').textContent).not.toContain(
      'private provider',
    );
    expect(mocks.fetch).toHaveBeenCalledWith(
      '/v2/images/app/tags/latest?expectedGeneration=7',
      { method: 'DELETE' },
    );
    expect(screen.getByText(`app@sha256:${sha}`)).toBeTruthy();
  });
});
