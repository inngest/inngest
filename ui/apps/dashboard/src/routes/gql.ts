/**
 * Demo-only GraphQL endpoint. Serves the fake App API (src/demo/mock) so the
 * dashboard can run against deterministic dummy data with VITE_API_URL pointed
 * at its own origin. Inert in the production build: returns 404 unless
 * VITE_DEMO_MODE is set, so this route never shadows anything in the real app.
 *
 * The mock server is imported lazily inside the handlers so its server-only
 * deps (graphql-yoga, @graphql-tools/*) are never bundled for the client.
 */
import { createFileRoute } from '@tanstack/react-router';

const isDemo = Boolean(import.meta.env.VITE_DEMO_MODE);

async function handle(request: Request): Promise<Response> {
  if (!isDemo) {
    return new Response('Not found', { status: 404 });
  }
  const { handleGraphQL } = await import('@/demo/mock/server');
  return handleGraphQL(request);
}

export const Route = createFileRoute('/gql')({
  server: {
    handlers: {
      GET: async ({ request }) => handle(request),
      POST: async ({ request }) => handle(request),
    },
  },
});
