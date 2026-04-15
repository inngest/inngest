import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import ExperimentsPage from '@/components/Experiments/ExperimentsPage';

export const Route = createFileRoute('/_authed/env/$envSlug/experiments/')({
  component: ExperimentsComponent,
});

function ExperimentsComponent() {
  const { envSlug } = Route.useParams();

  return (
    <ClientOnly>
      <ExperimentsPage environmentSlug={envSlug} />
    </ClientOnly>
  );
}
