import { FunctionAI } from '@/components/Functions/FunctionAI';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/functions/$slug/ai/',
)({
  component: RouteComponent,
});

function RouteComponent() {
  const { slug } = Route.useParams();
  return <FunctionAI functionSlug={slug} />;
}
