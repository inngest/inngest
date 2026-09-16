import { createFileRoute } from '@tanstack/react-router';

import { OAuthAuthorizationPage } from '../device';

type Search = {
  request?: string;
};

export const Route = createFileRoute('/_authed/oauth/authorize/')({
  component: AuthorizationRoute,
  validateSearch: (search: Record<string, unknown>): Search => ({
    request: typeof search.request === 'string' ? search.request : undefined,
  }),
});

function AuthorizationRoute() {
  const search = Route.useSearch();
  return <OAuthAuthorizationPage initialRequest={search.request} />;
}
