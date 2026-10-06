import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import SandboxesEmptyState from '@/components/Sandboxes/SandboxesEmptyState';
import { useEnvironment } from '@/components/Environments/environment-context';
import { SandboxesLayout } from '@/components/Sandboxes/SandboxesLayout';
import { SandboxesList } from '@/components/Sandboxes/SandboxesList';
import { ImagesPanel } from '@/components/Sandboxes/ImagesPanel';
import { getSandboxAPIEnabled } from '@/queries/server/featureFlags';

export const Route = createFileRoute('/_authed/env/$envSlug/sandboxes/')({
  component: SandboxesPage,
  loader: async () => ({
    sandboxAPIEnabled: await getSandboxAPIEnabled(),
  }),
});

function SandboxesPage() {
  const { sandboxAPIEnabled } = Route.useLoaderData();
  const environment = useEnvironment();
  const [view, setView] = useState<'sandboxes' | 'images'>('sandboxes');

  return (
    <>
      <Header breadcrumb={[{ text: 'Sandboxes' }]} />
      {sandboxAPIEnabled ? (
        <div className="flex min-h-0 flex-1 flex-col">
          <nav
            aria-label="Sandbox views"
            className="border-subtle flex gap-2 border-b px-6 py-2"
          >
            <Button
              label="Sandboxes"
              kind="secondary"
              appearance="ghost"
              aria-pressed={view === 'sandboxes'}
              onClick={() => setView('sandboxes')}
            />
            <Button
              label="Images"
              kind="secondary"
              appearance="ghost"
              aria-pressed={view === 'images'}
              onClick={() => setView('images')}
            />
          </nav>
          {view === 'sandboxes' ? (
            <SandboxesLayout key={environment.id}>
              <SandboxesList />
            </SandboxesLayout>
          ) : (
            <ImagesPanel key={environment.id} />
          )}
        </div>
      ) : (
        <div className="bg-canvasBase h-full w-full overflow-y-auto">
          <ClientOnly>
            <SandboxesEmptyState />
          </ClientOnly>
        </div>
      )}
    </>
  );
}
