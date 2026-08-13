import { createFileRoute } from '@tanstack/react-router';

import { SandboxDetail } from '@/components/Sandboxes/SandboxDetail';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/sandboxes/$sandboxID/',
)({
  component: SandboxDetailRoute,
});

function SandboxDetailRoute() {
  const { sandboxID } = Route.useParams();
  return <SandboxDetail sandboxID={sandboxID} />;
}
