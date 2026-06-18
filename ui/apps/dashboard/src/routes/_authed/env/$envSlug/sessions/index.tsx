import { Header } from '@inngest/components/Header/Header';
import { SessionsEmptyState } from '@inngest/components/Sessions/SessionsEmptyState';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import NotFound from '@/components/Error/NotFound';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { SessionsInfo } from '@/components/Sessions/SessionsInfo';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/sessions/')({
  component: SessionsPage,
});

function SessionsPage() {
  const { envSlug } = Route.useParams();
  const navigate = Route.useNavigate();
  const sessionsEnabled = useBooleanFlag('sessions-ui');

  if (sessionsEnabled.isReady && !sessionsEnabled.value) {
    return <NotFound />;
  }

  return (
    <>
      <Header breadcrumb={[{ text: 'Sessions' }]} infoIcon={<SessionsInfo />} />
      <ClientOnly>
        <SessionsEmptyState
          onSubmit={(sessionKey) =>
            navigate({ to: pathCreator.sessions({ envSlug, sessionKey }) })
          }
        />
      </ClientOnly>
    </>
  );
}
