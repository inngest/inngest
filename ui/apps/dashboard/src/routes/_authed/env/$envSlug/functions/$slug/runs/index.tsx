import { Runs } from '@/components/Runs/Runs';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/runs/',
)({
  component: FunctionRunsComponent,
});

function FunctionRunsComponent() {
  const { slug } = Route.useParams();
  return <Runs functionSlug={slug} scope="fn" />;
}
