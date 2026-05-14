import { useMemo } from 'react';

import { Button } from '@inngest/components/Button';
import { FunctionsTable } from '@inngest/components/Functions/FunctionsTable';
import { Header } from '@inngest/components/Header/Header';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { FunctionInfo } from '@/components/Functions/FunctionInfo';
import {
  useFunctionVolume,
  useFunctions,
} from '@/components/Functions/useFunctions';
import { AppFilterDocument } from '@/components/Runs/queries';
import { pathCreator } from '@/utils/urls';
import { ClientOnly, createFileRoute, useRouter } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/env/$envSlug/functions/')({
  component: FunctionPage,
});

function FunctionPage() {
  const { envSlug } = Route.useParams();
  const router = useRouter();
  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      function: (params: { functionSlug: string }) =>
        pathCreator.function({
          envSlug: envSlug,
          functionSlug: params.functionSlug,
        }),
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({
          envSlug: envSlug,
          eventName: params.eventName,
        }),
      app: (params: { externalAppID: string }) =>
        pathCreator.app({
          envSlug: envSlug,
          externalAppID: params.externalAppID,
        }),
    };
  }, [envSlug]);
  const getFunctions = useFunctions();
  const getFunctionVolume = useFunctionVolume();

  const env = useEnvironment();
  const [appsRes] = useQuery({
    query: AppFilterDocument,
    variables: { envSlug: env.slug },
  });
  const apps = useMemo(
    () =>
      appsRes.data?.env?.apps.map((app) => ({
        id: app.id,
        name: app.externalID,
      })),
    [appsRes.data]
  );

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Functions' }]}
        infoIcon={<FunctionInfo />}
      />
      <ClientOnly>
        <FunctionsTable
          key={envSlug}
          pathCreator={internalPathCreator}
          getFunctions={getFunctions}
          getFunctionVolume={getFunctionVolume}
          apps={apps}
          emptyActions={
            <>
              <Button
                appearance="outlined"
                label="Refresh"
                onClick={() => router.invalidate()}
                icon={<RiRefreshLine />}
                iconSide="left"
              />
              <Button
                label="Go to docs"
                href="https://www.inngest.com/docs/learn/inngest-functions"
                target="_blank"
                icon={<RiExternalLinkLine />}
                iconSide="left"
              />
            </>
          }
        />
      </ClientOnly>
    </>
  );
}
