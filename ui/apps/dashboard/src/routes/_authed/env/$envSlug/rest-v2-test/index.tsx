import { useMemo, useState } from 'react';
import { useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute } from '@tanstack/react-router';

import { useEnvironment } from '@/components/Environments/environment-context';

const defaultRunID = '01KR43RTM3PXNPNFT78Z82ZAP8';

type FetchState =
  | {
      status: 'idle' | 'loading';
    }
  | {
      body: unknown;
      status: 'success' | 'error';
      statusCode: number;
    };

export const Route = createFileRoute('/_authed/env/$envSlug/rest-v2-test/')({
  component: RestV2TestPage,
});

function RestV2TestPage() {
  const env = useEnvironment();
  const { getToken } = useAuth();
  const [runID, setRunID] = useState(defaultRunID);
  const [fetchState, setFetchState] = useState<FetchState>({ status: 'idle' });

  const requestURL = useMemo(() => {
    const url = new URL(
      `/v2/runs/${encodeURIComponent(runID)}/trace`,
      import.meta.env.VITE_API_URL,
    );
    return url.toString();
  }, [runID]);

  const fetchTrace = async () => {
    setFetchState({ status: 'loading' });

    const token = await getToken({ skipCache: true });
    if (!token) {
      setFetchState({
        body: { error: 'missing Clerk session token' },
        status: 'error',
        statusCode: 0,
      });
      return;
    }

    const headers = new Headers({
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    });

    if (env.slug !== 'production') {
      headers.set('X-Inngest-Env', env.slug);
    }

    const response = await fetch(requestURL, {
      credentials: 'include',
      headers,
    });
    const text = await response.text();
    const body = parseResponseBody(text);

    setFetchState({
      body,
      status: response.ok ? 'success' : 'error',
      statusCode: response.status,
    });
  };

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-4 px-6 py-8">
      <div>
        <h1 className="text-basis text-2xl font-medium">REST v2 auth test</h1>
        <p className="text-muted mt-1 text-sm">
          Fetches a run trace from REST v2 using the current Clerk session.
        </p>
      </div>

      <div className="border-subtle bg-canvasBase flex flex-col gap-3 rounded-md border p-4">
        <label className="text-basis flex flex-col gap-1 text-sm font-medium">
          Run ID
          <input
            className="border-subtle bg-canvasSubtle text-basis rounded-md border px-3 py-2 font-mono text-sm"
            onChange={(event) => setRunID(event.target.value)}
            value={runID}
          />
        </label>

        <div className="text-muted break-all font-mono text-xs">
          {requestURL}
        </div>
        <div className="text-muted text-xs">
          Env slug: <span className="font-mono">{env.slug}</span>
          {env.slug === 'production'
            ? ' (no X-Inngest-Env header sent)'
            : ' (sent as X-Inngest-Env)'}
        </div>

        <button
          className="bg-primary-moderate text-onContrast w-fit rounded-md px-3 py-2 text-sm font-medium disabled:opacity-50"
          disabled={fetchState.status === 'loading' || runID.trim() === ''}
          onClick={fetchTrace}
          type="button"
        >
          {fetchState.status === 'loading' ? 'Fetching...' : 'Fetch trace'}
        </button>
      </div>

      {'statusCode' in fetchState && (
        <div className="border-subtle bg-canvasBase rounded-md border p-4">
          <div className="text-basis mb-3 text-sm font-medium">
            HTTP {fetchState.statusCode} {fetchState.status}
          </div>
          <pre className="bg-canvasSubtle text-basis max-h-[60vh] overflow-auto rounded-md p-3 text-xs">
            {JSON.stringify(fetchState.body, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function parseResponseBody(text: string): unknown {
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}
