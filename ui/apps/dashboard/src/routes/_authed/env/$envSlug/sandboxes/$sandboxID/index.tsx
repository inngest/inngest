import { createFileRoute, redirect } from '@tanstack/react-router';

import { SandboxDetail } from '@/components/Sandboxes/SandboxDetail';
import { getSandboxAPIEnabled } from '@/queries/server/featureFlags';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/sandboxes/$sandboxID/',
)({
  component: SandboxDetailRoute,
  loader: async ({ params }) => {
    if (!(await getSandboxAPIEnabled())) {
      throw redirect({
        to: pathCreator.sandboxes({ envSlug: params.envSlug }),
      });
    }
  },
});

function SandboxDetailRoute() {
  const { sandboxID } = Route.useParams();
  return <SandboxDetail sandboxID={sandboxID} />;
}
