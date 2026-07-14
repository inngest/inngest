import { useEffect } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import { useDefaultEnvironment } from '@/queries';

export const Route = createFileRoute('/')({
  component: Home,
});

function Home() {
  const navigate = useNavigate();
  const [{ data: defaultEnv, fetching }] = useDefaultEnvironment();

  // Forward into the environment subtree as soon as the default-env query
  // settles. We intentionally don't gate on the navigation-version flag here:
  // it only resolves once LaunchDarkly identifies a signed-in user, so gating
  // on it would deadlock signed-out visitors on this skeleton and never let the
  // _authed guard redirect them to sign-in. The /env/$envSlug index route makes
  // the v1/v2 decision once the user is authenticated.
  useEffect(() => {
    if (fetching) {
      return;
    }

    const envSlug = defaultEnv?.slug ?? 'production';
    void navigate({
      to: '/env/$envSlug',
      params: { envSlug },
      replace: true,
    });
  }, [defaultEnv?.slug, fetching, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <Skeleton className="h-8 w-40" />
    </div>
  );
}
