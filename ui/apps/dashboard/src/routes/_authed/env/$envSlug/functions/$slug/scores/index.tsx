import { FunctionScores } from '@/components/Functions/FunctionScores';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/scores/',
)({
  component: RouteComponent,
});

function RouteComponent() {
  const { slug } = Route.useParams();
  return <FunctionScores functionSlug={slug} />;
}
