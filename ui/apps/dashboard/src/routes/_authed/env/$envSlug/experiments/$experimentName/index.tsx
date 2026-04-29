import { createFileRoute, redirect } from '@tanstack/react-router';

// Legacy route — experiments now live nested under their function so two
// functions sharing an experiment name can be disambiguated. The old URL
// shape (/experiments/$experimentName) doesn't carry the function, so
// redirect any deep links / bookmarks back to the all-experiments list and
// let the user re-pick the row they want.
export const Route = createFileRoute(
  '/_authed/env/$envSlug/experiments/$experimentName/',
)({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: '/env/$envSlug/experiments',
      params: { envSlug: params.envSlug },
    });
  },
});
