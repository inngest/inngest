import { useMemo } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { SessionResults } from '@inngest/components/Sessions/SessionResults';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { useEnvironment } from '@/components/Environments/environment-context';
import { SessionsInfo } from '@/components/Sessions/SessionsInfo';
import { useSessions } from '@/components/Sessions/useSessions';
import { pathCreator } from '@/utils/urls';
import { useAccountFeatures } from '@/utils/useAccountFeatures';

type SessionsResultsSearchParams = {
  last?: string;
  start?: string;
  end?: string;
};

export const Route = createFileRoute(
  '/_authed/env/$envSlug/sessions/$sessionKey/',
)({
  component: SessionResultsPage,
  validateSearch: (
    search: Record<string, unknown>,
  ): SessionsResultsSearchParams => ({
    last: typeof search?.last === 'string' ? search.last : undefined,
    start: typeof search?.start === 'string' ? search.start : undefined,
    end: typeof search?.end === 'string' ? search.end : undefined,
  }),
});

function SessionResultsPage() {
  const { envSlug, sessionKey } = Route.useParams();
  const { last, start, end } = Route.useSearch();
  const navigate = Route.useNavigate();
  const envID = useEnvironment().id;
  const features = useAccountFeatures();
  const getSessions = useSessions();
  const decodedSessionKey = decodeURIComponent(sessionKey);

  const internalPathCreator = useMemo(
    () => ({
      session: (params: { sessionKey: string; sessionId: string }) =>
        pathCreator.session({
          envSlug,
          sessionKey: params.sessionKey,
          sessionId: params.sessionId,
        }),
      function: (params: { functionSlug: string }) =>
        pathCreator.function({ envSlug, functionSlug: params.functionSlug }),
    }),
    [envSlug],
  );

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Sessions', href: pathCreator.sessions({ envSlug }) },
          { text: decodedSessionKey },
        ]}
        infoIcon={<SessionsInfo />}
      />
      <ClientOnly>
        <SessionResults
          envID={envID}
          maxRangeDays={features.data?.history ?? 7}
          sessionKey={decodedSessionKey}
          last={last}
          start={start}
          end={end}
          pathCreator={internalPathCreator}
          getSessions={getSessions}
          onEditSearch={() =>
            navigate({ to: pathCreator.sessions({ envSlug }) })
          }
        />
      </ClientOnly>
    </>
  );
}
