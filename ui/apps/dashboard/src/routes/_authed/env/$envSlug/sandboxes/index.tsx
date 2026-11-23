import { Header } from '@inngest/components/Header/Header';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { SandboxesAccess } from '@/components/Sandboxes/SandboxesAccess';

export const Route = createFileRoute('/_authed/env/$envSlug/sandboxes/')({
  component: SandboxesPage,
});

function SandboxesPage() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Sandboxes' }]} />
      <div className="bg-canvasBase h-full w-full overflow-y-auto">
        <ClientOnly>
          <SandboxesAccess />
        </ClientOnly>
      </div>
    </>
  );
}
