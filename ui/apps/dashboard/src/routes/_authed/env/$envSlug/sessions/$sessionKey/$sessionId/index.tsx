import { Header } from '@inngest/components/Header/Header';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { SessionRuns } from '@/components/Sessions/SessionRuns';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/sessions/$sessionKey/$sessionId/',
)({ component: SessionDetailPage });

function SessionDetailPage() {
  const { envSlug, sessionKey, sessionId } = Route.useParams();
  const sessionsEnabled = useBooleanFlag('sessions-ui');
  const name = decodeURIComponent(sessionKey);
  const id = decodeURIComponent(sessionId);

  if (sessionsEnabled.isReady && !sessionsEnabled.value) {
    return <NotFound />;
  }

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Sessions', href: pathCreator.sessions({ envSlug }) },
          {
            text: name,
            href: pathCreator.sessions({ envSlug, sessionKey: name }),
          },
          { text: id },
        ]}
      />
      <ClientOnly>
        <SessionRuns sessionKey={name} sessionId={id} />
      </ClientOnly>
    </>
  );
}
