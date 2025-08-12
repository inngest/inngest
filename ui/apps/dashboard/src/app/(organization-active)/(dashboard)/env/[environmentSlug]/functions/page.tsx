'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/Button';
import { FunctionsTable } from '@inngest/components/Functions/FunctionsTable';
import { Header } from '@inngest/components/Header/Header';
import { RiExternalLinkLine, RiRefreshLine } from '@remixicon/react';

import { FunctionInfo } from '@/components/Functions/FunctionInfo';
import { useFunctionVolume, useFunctions } from '@/components/Functions/useFunctions';
import { pathCreator } from '@/utils/urls';

export default function FunctionPage({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  const router = useRouter();
  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      function: (params: { functionSlug: string }) =>
        pathCreator.function({ envSlug: envSlug, functionSlug: params.functionSlug }),
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({ envSlug: envSlug, eventName: params.eventName }),
      app: (params: { externalAppID: string }) =>
        pathCreator.app({ envSlug: envSlug, externalAppID: params.externalAppID }),
    };
  }, [envSlug]);
  const getFunctions = useFunctions();
  const getFunctionVolume = useFunctionVolume();

  return (
    <>
      <Header breadcrumb={[{ text: 'Functions' }]} infoIcon={<FunctionInfo />} />
      <FunctionsTable
        pathCreator={internalPathCreator}
        getFunctions={getFunctions}
        getFunctionVolume={getFunctionVolume}
        emptyActions={
          <>
            <Button
              appearance="outlined"
              label="Refresh"
              onClick={() => router.refresh()}
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
    </>
  );
}
