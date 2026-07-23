import { lazy } from 'react';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { useFunction } from '@/queries/functions';

const FunctionAIPanel = lazy(() =>
  import('@/components/AIOverview/FunctionPanel').then((m) => ({
    default: m.FunctionAIPanel,
  })),
);

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/ai/',
)({
  component: RouteComponent,
});

function RouteComponent() {
  const { slug } = Route.useParams();
  const functionSlug = decodeURIComponent(slug);
  const [{ data, fetching }] = useFunction({ functionSlug });
  const functionID = data?.workspace.workflow?.id;

  if (fetching || !functionID) {
    return null;
  }

  return (
    <ClientOnly>
      <FunctionAIPanel functionID={functionID} />
    </ClientOnly>
  );
}
