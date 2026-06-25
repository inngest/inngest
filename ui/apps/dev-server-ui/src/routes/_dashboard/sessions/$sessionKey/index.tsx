import { useMemo } from 'react';
import { Header } from '@inngest/components/Header/Header';
import { SessionResults } from '@inngest/components/Sessions/SessionResults';
import { ClientOnly, createFileRoute } from '@tanstack/react-router';

import { useSessions } from '@/components/Sessions/useSessions';
import { pathCreator } from '@/utils/pathCreator';

type SessionResultsSearchParams = {
  last?: string;
  start?: string;
  end?: string;
};

export const Route = createFileRoute('/_dashboard/sessions/$sessionKey/')({
  component: SessionResultsPage,
  validateSearch: (
    search: Record<string, unknown>,
  ): SessionResultsSearchParams => ({
    last: typeof search?.last === 'string' ? search.last : undefined,
    start: typeof search?.start === 'string' ? search.start : undefined,
    end: typeof search?.end === 'string' ? search.end : undefined,
  }),
});

function SessionResultsPage() {
  const { sessionKey } = Route.useParams();
  const { last, start, end } = Route.useSearch();
  const navigate = Route.useNavigate();
  const getSessions = useSessions();
  const decodedSessionKey = decodeURIComponent(sessionKey);

  const internalPathCreator = useMemo(
    () => ({
      session: (params: { sessionKey: string; sessionId: string }) =>
        pathCreator.session({
          sessionKey: params.sessionKey,
          sessionId: params.sessionId,
        }),
      function: (params: { functionSlug: string }) =>
        pathCreator.function({ functionSlug: params.functionSlug }),
    }),
    [],
  );

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Sessions', href: pathCreator.sessions({}) },
          { text: decodedSessionKey },
        ]}
      />
      <ClientOnly>
        <SessionResults
          envID="local"
          maxRangeDays={7}
          sessionKey={decodedSessionKey}
          last={last}
          start={start}
          end={end}
          pathCreator={internalPathCreator}
          getSessions={getSessions}
          onEditSearch={() => navigate({ to: pathCreator.sessions({}) })}
        />
      </ClientOnly>
    </>
  );
}
