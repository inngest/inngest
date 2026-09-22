import { useEffect, useState, type ReactNode } from 'react';
import { Link, useLocation } from '@tanstack/react-router';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import {
  getAPIOrigin,
  portStorageKey,
  probeDevServer,
  validPort,
  type ConnectionStatus,
} from '@/utils/devServer';

const messages: Record<ConnectionStatus, string> = {
  checking:
    'Checking for your local Dev Server. Allow local network access if your browser asks.',
  connected: 'Your local Dev Server is connected.',
  unreachable:
    'We could not connect. Start the Dev Server below, or check that your browser allows local network access.',
  blocked:
    'Your browser blocked local network access. Allow it in this site’s permissions, then try again.',
  unauthorized:
    'This server requires authentication. Check its configuration or use its local UI.',
  'wrong-service':
    'This port did not return Inngest server information. Check the port or update your Inngest CLI.',
  'api-unavailable':
    'The server’s UI API is unavailable. Start the Dev Server without --no-ui, or update your Inngest CLI.',
};

export function HostedConnection({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<ConnectionStatus>('checking');
  const [attempt, setAttempt] = useState(0);
  const [port, setPort] = useState('8288');
  const [origin, setOrigin] = useState('http://localhost:8288');
  const [copyMessage, setCopyMessage] = useState('');
  const [portError, setPortError] = useState('');
  const pathname = useLocation({ select: (location) => location.pathname });
  const isHome =
    pathname === '/' || pathname === '/dev' || pathname === '/dev/';

  useEffect(() => {
    setOrigin(getAPIOrigin());
    setPort(new URL(getAPIOrigin()).port || '80');
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    let controller: AbortController;
    async function check() {
      controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 10_000);
      const next = await probeDevServer(controller.signal);
      clearTimeout(timeout);
      if (stopped) return;
      setStatus(next);
      if (next === 'connected' || next === 'unreachable') {
        timer = setTimeout(check, next === 'connected' ? 10_000 : 3_000);
      }
    }
    void check();
    return () => {
      stopped = true;
      controller?.abort();
      clearTimeout(timer);
    };
  }, [attempt]);

  if (status === 'connected' && !isHome) return <>{children}</>;

  return (
    <main className="bg-canvasBase text-basis min-h-screen px-5 py-10 sm:py-16">
      <div className="mx-auto max-w-2xl">
        <a href="/" aria-label="Inngest home" className="inline-block">
          <InngestLogo width={120} />
        </a>
        <h1 className="mt-10 text-3xl font-semibold sm:text-4xl">
          Inngest Dev Server
        </h1>
        <p className="text-subtle mt-4 text-lg">
          Build and debug durable workflows locally. Send test events, inspect
          function runs, and explore execution traces from your browser.
        </p>
        <p className="text-subtle mt-3">
          This UI connects directly to the Inngest Dev Server on this device.
          Your apps and functions run on your machine.
        </p>
        <section
          aria-labelledby="connection-heading"
          className="border-subtle bg-canvasSubtle mt-8 rounded-lg border p-5 sm:p-6"
        >
          <h2 id="connection-heading" className="text-lg font-medium">
            {status === 'connected'
              ? 'Connected to localhost'
              : 'Connect your local server'}
          </h2>
          <p role="status" aria-live="polite" className="text-subtle mt-2">
            {messages[status]}
          </p>
          <p className="text-muted mt-2 break-all text-sm">{origin}</p>
          {status === 'connected' ? (
            <Link
              to="/runs"
              className="bg-primary-moderate mt-5 inline-flex rounded px-4 py-2 font-medium"
            >
              Open dashboard
            </Link>
          ) : (
            <>
              <p className="mt-5">Run this command in your terminal:</p>
              <pre className="bg-canvasBase border-subtle mt-3 overflow-x-auto rounded border p-4 text-sm">
                <code>npx --ignore-scripts=false inngest-cli@latest dev</code>
              </pre>
              <div className="mt-4 flex flex-wrap gap-3">
                <button
                  type="button"
                  className="border-subtle rounded border px-4 py-2"
                  onClick={async () => {
                    try {
                      await navigator.clipboard.writeText(
                        'npx --ignore-scripts=false inngest-cli@latest dev',
                      );
                      setCopyMessage('Command copied.');
                    } catch {
                      setCopyMessage('Select the command above to copy it.');
                    }
                  }}
                >
                  Copy command
                </button>
                <button
                  type="button"
                  className="border-subtle rounded border px-4 py-2"
                  onClick={() => {
                    setStatus('checking');
                    setAttempt((value) => value + 1);
                  }}
                >
                  Try again
                </button>
              </div>
              <p role="status" className="text-muted mt-2 text-sm">
                {copyMessage}
              </p>
            </>
          )}
          <details className="mt-6">
            <summary className="cursor-pointer">Use a different port</summary>
            <form
              className="mt-3 flex flex-wrap items-end gap-3"
              onSubmit={(event) => {
                event.preventDefault();
                if (!validPort(port)) {
                  setPortError('Enter a port from 1 to 65535.');
                  return;
                }
                try {
                  window.localStorage.setItem(
                    portStorageKey,
                    String(Number(port)),
                  );
                  window.location.reload();
                } catch {
                  setPortError('Allow site storage to save a different port.');
                }
              }}
            >
              <label className="text-sm">
                Localhost port
                <input
                  type="number"
                  min="1"
                  max="65535"
                  required
                  value={port}
                  onChange={(event) => setPort(event.target.value)}
                  className="bg-canvasBase border-subtle mt-1 block w-32 rounded border p-2"
                />
              </label>
              <button
                className="border-subtle rounded border px-4 py-2"
                type="submit"
              >
                Connect
              </button>
              {portError && <p role="alert">{portError}</p>}
            </form>
          </details>
        </section>
        <div className="mt-6 flex flex-wrap gap-5 text-sm">
          <a
            className="text-link underline"
            href="/docs/local-development?ref=dev"
          >
            Setup guide
          </a>
          <a className="text-link underline" href={origin}>
            Open local UI
          </a>
        </div>
        <p className="text-muted mt-6 text-sm">
          Keep the Dev Server running while you work. On a phone or tablet,
          localhost refers to that device, not your computer.
        </p>
      </div>
    </main>
  );
}
