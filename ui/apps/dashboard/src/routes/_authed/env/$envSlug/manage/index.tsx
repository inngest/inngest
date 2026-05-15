import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/env/$envSlug/manage/')({
  loader: ({ params }) => {
    redirect({
      to: '/env/$envSlug/manage/$ingestKeys',
      params: { envSlug: params.envSlug, ingestKeys: 'keys' },
      throw: true,
    });
  },
});
