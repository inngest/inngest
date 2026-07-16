import { useState } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { SessionKeys } from '@inngest/components/Sessions/SessionKeys';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import FeedbackFloatingButton from '@/components/Feedback/FeedbackFloatingButton';
import { SessionsInfo } from '@/components/Sessions/SessionsInfo';
import { useSessionKeys } from '@/components/Sessions/useSessionKeys';
import {
  trackEmptyStateDocsLinkOpened,
  trackEmptyStateExampleCopied,
  trackEmptyStatePromptCopied,
  trackEmptyStateViewed,
} from '@/utils/analyticsEvents';
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
          onEmptyStateViewed={() =>
            trackEmptyStateViewed({ feature: 'sessions' })
          }
          onEmptyStateDocsLinkClick={() =>
            trackEmptyStateDocsLinkOpened({ feature: 'sessions' })
          }
          onEmptyStatePromptCopy={() =>
            trackEmptyStatePromptCopied({ feature: 'sessions' })
          }
          onEmptyStateExampleCopy={() =>
            trackEmptyStateExampleCopied({ feature: 'sessions' })
          }
        />
      </ClientOnly>
      <FeedbackFloatingButton />
    </>
  );
}
