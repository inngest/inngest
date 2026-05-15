import { CancellationTable } from '@/components/Functions/CancellationTable';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/cancellations/',
)({
  component: RouteComponent,
});

function RouteComponent() {
  const { slug, envSlug } = Route.useParams();

  return <CancellationTable envSlug={envSlug} fnSlug={slug} />;
}
