import { useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { SessionKeys } from '@inngest/components/Sessions/SessionKeys';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import FeedbackFloatingButton from '@/components/Feedback/FeedbackFloatingButton';
import { SessionsEmptyState } from '@/components/Sessions/SessionsEmptyState';
import { SessionsInfo } from '@/components/Sessions/SessionsInfo';
import { trackEmptyStateDocsLinkOpened } from '@/utils/analyticsEvents';
import { useSessionKeys } from '@/components/Sessions/useSessionKeys';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/sessions/')({
  component: SessionsPage,
});

function SessionsPage() {
  const { envSlug } = Route.useParams();
  const navigate = Route.useNavigate();
  const [search, setSearch] = useState('');
  const { data, error, isPending, isFetching, refetch } =
    useSessionKeys(search);

  function navigateToSessionKey(sessionKey: string) {
    navigate({ to: pathCreator.sessions({ envSlug, sessionKey }) });
  }

  const sessionKeys = data ?? [];
  const showEmptyState = !error && !search.trim() && sessionKeys.length === 0;

  return (
    <>
      <Header breadcrumb={[{ text: 'Sessions' }]} infoIcon={<SessionsInfo />} />
      <ClientOnly>
        {showEmptyState ? (
          <SessionsEmptyState
            onDocsLinkClick={() =>
              trackEmptyStateDocsLinkOpened({ feature: 'sessions' })
            }
          />
        ) : (
          <SessionKeys
            sessionKeys={sessionKeys}
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
        )}
      </ClientOnly>
      <FeedbackFloatingButton />
    </>
  );
}
