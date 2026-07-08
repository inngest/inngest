import { Header } from '@inngest/components/Header/Header';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { SessionRuns } from '@/components/Sessions/SessionRuns';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/sessions/$sessionKey/$sessionId/',
)({ component: SessionDetailPage });

function SessionDetailPage() {
  const { envSlug, sessionKey, sessionId } = Route.useParams();
  const name = decodeURIComponent(sessionKey);
  const id = decodeURIComponent(sessionId);

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
