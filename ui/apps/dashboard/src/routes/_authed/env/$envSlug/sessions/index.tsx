import { useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { SessionKeys } from '@inngest/components/Sessions/SessionKeys';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { SessionsInfo } from '@/components/Sessions/SessionsInfo';
import { useSessionKeys } from '@/components/Sessions/useSessionKeys';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/sessions/')({
  component: SessionsPage,
});

function SessionsPage() {
  const { envSlug } = Route.useParams();
  const navigate = Route.useNavigate();
  const sessionsEnabled = useBooleanFlag('sessions-ui');
  const [search, setSearch] = useState('');
  const { data, error, isPending, isFetching, refetch } =
    useSessionKeys(search);

  function navigateToSessionKey(sessionKey: string) {
    navigate({ to: pathCreator.sessions({ envSlug, sessionKey }) });
  }

  if (sessionsEnabled.isReady && !sessionsEnabled.value) {
    return <NotFound />;
  }

  return (
    <>
      <Header breadcrumb={[{ text: 'Sessions' }]} infoIcon={<SessionsInfo />} />
      <ClientOnly>
        <SessionKeys
          sessionKeys={data ?? []}
          isLoading={isPending || isFetching}
          search={search}
          error={error}
          onSearchChange={setSearch}
          onSubmitSearch={navigateToSessionKey}
          onRefresh={() => refetch()}
          onSelectSessionKey={navigateToSessionKey}
          getSessionKeyHref={(sessionKey) =>
            pathCreator.sessions({ envSlug, sessionKey })
          }
        />
      </ClientOnly>
    </>
  );
}
