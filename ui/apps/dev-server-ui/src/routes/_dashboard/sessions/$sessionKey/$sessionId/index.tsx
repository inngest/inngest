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
  const decodedSessionKey = decodeURIComponent(sessionKey);
  const decodedSessionId = decodeURIComponent(sessionId);

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Sessions', href: pathCreator.sessions({}) },
          {
            text: decodedSessionKey,
            href: pathCreator.sessions({ sessionKey: decodedSessionKey }),
          },
          { text: decodedSessionId },
        ]}
      />
      <ClientOnly>
        <SessionRuns
          sessionKey={decodedSessionKey}
          sessionId={decodedSessionId}
        />
      </ClientOnly>
    </>
  );
}
