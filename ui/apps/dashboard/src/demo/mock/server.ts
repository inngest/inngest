/**
 * graphql-yoga instance serving the fake App API for the demo build. Mounted at
 * /gql by src/routes/gql.ts, which both GraphQL transports (urql + graphql-
 * request) target when VITE_API_URL points at the app's own origin.
 */
import { createYoga } from 'graphql-yoga';

import { getMockSchema } from './schema';
import { resetScalarCounter } from './scalars';

let yoga: ReturnType<typeof createYoga> | undefined;

export function getYoga() {
  if (yoga) return yoga;
  yoga = createYoga({
    schema: getMockSchema(),
    graphqlEndpoint: '/gql',
    // The demo has no auth; keep GraphiQL available for exploration.
    graphiql: true,
    landingPage: false,
    cors: false,
    plugins: [
      {
        // Reset the deterministic scalar sequence per request so identical
        // queries yield identical responses.
        onRequest() {
          resetScalarCounter();
        },
      },
    ],
  });
  return yoga;
}

export async function handleGraphQL(request: Request): Promise<Response> {
  // `.fetch` is yoga's Request -> Response adapter for fetch environments.
  const res = await getYoga().fetch(request);
  // Yoga returns its own ponyfilled Response whose body the host runtime
  // (srvx/Nitro) doesn't stream — materialize it into a native Response so the
  // body and headers are forwarded intact.
  const body = await res.arrayBuffer();
  return new Response(body, {
    status: res.status,
    statusText: res.statusText,
    headers: res.headers,
  });
}
