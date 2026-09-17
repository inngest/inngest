import { Header } from '@inngest/components/Header/Header';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { SessionRuns } from '@/components/Sessions/SessionRuns';
import { pathCreator } from '@/utils/pathCreator';

export const Route = createFileRoute(
  '/_dashboard/sessions/$sessionKey/$sessionId/',
)({
  component: SessionRunsPage,
});

function SessionRunsPage() {
  const { sessionKey, sessionId } = Route.useParams();

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Sessions', href: pathCreator.sessions({}) },
          {
            text: sessionKey,
            href: pathCreator.sessions({ sessionKey }),
          },
          { text: sessionId },
        ]}
      />
      <ClientOnly>
        <SessionRuns sessionKey={sessionKey} sessionId={sessionId} />
      </ClientOnly>
    </>
  );
}
