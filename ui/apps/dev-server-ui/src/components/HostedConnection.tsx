import { useEffect, useState, type ReactNode } from 'react';
import { useLocation } from '@tanstack/react-router';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { TooltipProvider } from '@inngest/components/Tooltip';

import Layout from '@/components/Layout/Layout';
import StoreProvider from '@/components/StoreProvider';
import {
  getAPIOrigin,
  portStorageKey,
  probeDevServer,
  validPort,
  type ConnectionStatus,
} from '@/utils/devServer';

const command = 'npx --ignore-scripts=false inngest-cli@latest dev';

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
    const apiOrigin = getAPIOrigin();
    setOrigin(apiOrigin);
    setPort(new URL(apiOrigin).port || '80');
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
    <StoreProvider>
      <TooltipProvider delayDuration={0}>
        <Layout connected={status === 'connected'}>
          <div className="flex h-full flex-col overflow-y-scroll">
            <Header breadcrumb={[{ text: 'Dev Server' }]} />
            <main className="mx-auto w-full max-w-4xl px-6 pb-4 pt-16">
              <h1 className="mb-1 text-xl">
                {status === 'connected'
                  ? 'Your Dev Server is ready'
                  : 'Connect your Dev Server'}
              </h1>
              <p className="text-subtle text-sm">
                Build and debug durable workflows locally. Send test events,
                inspect function runs, and explore execution traces from your
                browser.
              </p>
              <div className="bg-disabled my-4 flex flex-wrap items-center justify-between gap-2 rounded p-4">
                <div>
                  <p
                    role="status"
                    aria-live="polite"
                    className="text-subtle text-sm"
                  >
                    {messages[status]}
                  </p>
                  <p className="text-muted mt-1 break-all text-sm">{origin}</p>
                </div>
                {status === 'connected' && (
                  <Button kind="primary" label="View runs" to="/runs" />
                )}
              </div>
              {status !== 'connected' && (
                <section
                  aria-labelledby="start-server-heading"
                  className="mt-8"
                >
                  <h2 id="start-server-heading" className="mb-1 text-lg">
                    Start the Dev Server
                  </h2>
                  <p className="text-subtle text-sm">
                    Run this command in your terminal, then return to this page.
                    Your apps and functions run on your machine.
                  </p>
                  <pre className="bg-canvasSubtle border-subtle mt-4 overflow-x-auto rounded border p-4 text-sm">
                    <code>{command}</code>
                  </pre>
                  <div className="mt-4 flex flex-wrap gap-3">
                    <button
                      type="button"
                      className="border-subtle rounded border px-4 py-2 text-sm"
                      onClick={async () => {
                        try {
                          await navigator.clipboard.writeText(command);
                          setCopyMessage('Command copied.');
                        } catch {
                          setCopyMessage(
                            'Select the command above to copy it.',
                          );
                        }
                      }}
                    >
                      Copy command
                    </button>
                    <button
                      type="button"
                      className="border-subtle rounded border px-4 py-2 text-sm"
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
                </section>
              )}
              <details className="mt-8">
                <summary className="cursor-pointer text-sm">
                  Use a different port
                </summary>
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
                      setPortError(
                        'Allow site storage to save a different port.',
                      );
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
                    className="border-subtle rounded border px-4 py-2 text-sm"
                    type="submit"
                  >
                    Connect
                  </button>
                  {portError && <p role="alert">{portError}</p>}
                </form>
              </details>
              <div className="mt-8 flex flex-wrap gap-5 text-sm">
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
                Keep the Dev Server running while you work. On a phone or
                tablet, localhost refers to that device, not your computer.
              </p>
            </main>
          </div>
        </Layout>
      </TooltipProvider>
    </StoreProvider>
  );
}
