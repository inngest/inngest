import { useEffect } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import { InfraDashboard } from '@/components/InfraDashboard/InfraDashboard';
import { useNavigationV2State } from '@/components/Layout/useNavigationV2';

export const Route = createFileRoute('/_authed/env/$envSlug/')({
  component: EnvHome,
});

function EnvHome() {
  const { envSlug } = Route.useParams();
  const navigate = useNavigate();
  const navigation = useNavigationV2State();

  useEffect(() => {
    if (!navigation.isReady || navigation.value) {
      return;
    }

    void navigate({
      to: '/env/$envSlug/apps',
      params: { envSlug },
      replace: true,
    });
  }, [envSlug, navigate, navigation.isReady, navigation.value]);

  if (!navigation.isReady || !navigation.value) {
    return (
      <div className="flex min-h-full items-center justify-center p-8">
        <Skeleton className="h-8 w-40" />
      </div>
    );
  }

  return <InfraDashboard />;
}
