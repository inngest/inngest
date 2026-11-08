import { useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { SessionKeys } from '@inngest/components/Sessions/SessionKeys';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { useSessionKeys } from '@/components/Sessions/useSessionKeys';
import { pathCreator } from '@/utils/pathCreator';

export const Route = createFileRoute('/_dashboard/sessions/')({
  component: SessionsPage,
});

function SessionsPage() {
  const navigate = Route.useNavigate();
  const [search, setSearch] = useState('');
  const {
    data,
    error,
    isPending,
    isFetching,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
    refetch,
  } = useSessionKeys(search);

  function navigateToSessionKey(sessionKey: string) {
    navigate({ to: pathCreator.sessions({ sessionKey }) });
  }

  return (
    <>
      <Header breadcrumb={[{ text: 'Sessions' }]} />
      <ClientOnly>
        <SessionKeys
          sessionKeys={data ?? []}
          isLoading={isPending || (isFetching && !isFetchingNextPage)}
          search={search}
          error={error}
          onSearchChange={setSearch}
          onSubmitSearch={navigateToSessionKey}
          onRefresh={() => refetch()}
          hasMore={hasNextPage}
          isLoadingMore={isFetchingNextPage}
          onLoadMore={() => fetchNextPage()}
          onSelectSessionKey={navigateToSessionKey}
          getSessionKeyHref={(sessionKey) =>
            pathCreator.sessions({ sessionKey })
          }
        />
      </ClientOnly>
    </>
  );
}
