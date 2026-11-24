import { cleanup, fireEvent, render, waitFor } from '@testing-library/react';
import { Provider } from 'react-redux';
import { toast } from 'sonner';
import { afterEach, expect, it, vi } from 'vitest';

import { client } from '@/store/baseApi';
import { AppMethod, type GetAppsQuery } from '@/store/generated';
import { store } from '@/store/store';
import UpdateApp from './UpdateApp';

vi.mock('@/store/devApi', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/store/devApi')>()),
  useInfoQuery: () => ({ data: { isSingleNodeService: false } }),
}));

const app: GetAppsQuery['apps'][number] = {
  id: 'test-app',
  name: 'Test app',
  appVersion: null,
  sdkLanguage: 'js',
  sdkVersion: '1.0.0',
  framework: null,
  url: 'http://localhost:3000/api/inngest',
  error: null,
  connected: true,
  functionCount: 0,
  autodiscovered: false,
  method: AppMethod.Serve,
  sdkFeatureReadiness: { aiMetadataExtraction: null, extendedTraces: null },
  functions: [],
};

const originalFetch = client.requestConfig.fetch;

afterEach(() => {
  cleanup();
  client.requestConfig.fetch = originalFetch;
  vi.restoreAllMocks();
});

it.each([false, true])(
  'reports the update as successful only if the server accepts it (accepted=%s)',
  async (accepted) => {
    const newUrl = 'http://localhost:3001/api/inngest';
    const fetch = vi.fn<typeof globalThis.fetch>(async () =>
      Response.json(
        accepted
          ? { data: { updateApp: { id: app.id, url: newUrl } } }
          : { errors: [{ message: 'App update rejected' }] },
      ),
    );
    client.requestConfig.fetch = fetch;
    const success = vi.spyOn(toast, 'success').mockImplementation(() => '');
    const failure = vi.spyOn(toast, 'error').mockImplementation(() => '');
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});

    const { getByRole } = render(
      <Provider store={store}>
        <UpdateApp app={app} />
      </Provider>,
    );
    fireEvent.change(getByRole('textbox'), { target: { value: newUrl } });

    if (accepted) {
      await waitFor(() => expect(success).toHaveBeenCalledOnce(), {
        timeout: 2500,
      });
      expect(failure).not.toHaveBeenCalled();
    } else {
      await waitFor(() => expect(failure).toHaveBeenCalledOnce(), {
        timeout: 2500,
      });
      expect(success).not.toHaveBeenCalled();
    }
    expect(fetch).toHaveBeenCalledOnce();
    const [[url, options]] = fetch.mock.calls;
    expect(url).toBe('/v0/gql');
    expect(options?.method).toBe('POST');
    expect(JSON.parse(String(options?.body))).toMatchObject({
      operationName: 'UpdateApp',
      variables: { input: { id: app.id, url: newUrl } },
    });
  },
);
