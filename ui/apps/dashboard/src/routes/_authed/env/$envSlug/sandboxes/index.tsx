import { Header } from '@inngest/components/Header/Header';
import { createFileRoute } from '@tanstack/react-router';

import { SandboxesList } from '@/components/Sandboxes/SandboxesList';

export const Route = createFileRoute('/_authed/env/$envSlug/sandboxes/')({
  component: SandboxesPage,
});

function SandboxesPage() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Sandboxes' }]} />
      <SandboxesList />
    </>
  );
}
