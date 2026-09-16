import { Header } from '@inngest/components/Header/Header';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import SandboxesEmptyState from '@/components/Sandboxes/SandboxesEmptyState';
import { useEnvironment } from '@/components/Environments/environment-context';
import { SandboxesLayout } from '@/components/Sandboxes/SandboxesLayout';
import { SandboxesList } from '@/components/Sandboxes/SandboxesList';
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

  return (
    <>
      <Header breadcrumb={[{ text: 'Sandboxes' }]} />
      {sandboxAPIEnabled ? (
        <SandboxesLayout key={environment.id}>
          <SandboxesList />
        </SandboxesLayout>
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
