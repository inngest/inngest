import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/env/$slug/')({
  component: Page,
});

function Page() {
  const { slug } = Route.useParams();
  return `env ${slug}`;
}
