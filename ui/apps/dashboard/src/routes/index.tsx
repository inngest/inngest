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
  // settles so the _authed guard can handle signed-out visitors.
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
