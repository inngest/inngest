'use client';

import { useEnvironment } from '@/queries';
import { AppCard } from './AppCard';
import { FunctionList } from './FunctionList';
import { GitCard } from './GitCard';
import { useApp } from './useApp';

type Props = {
  params: {
    environmentSlug: string;
    externalID: string;
  };
};

export default function Page({ params: { environmentSlug, externalID } }: Props) {
  const [envRes] = useEnvironment({ environmentSlug });
  if (envRes.error) {
    throw envRes.error;
  }

  const appRes = useApp({
    envID: envRes.data?.id ?? '',
    externalAppID: externalID,
    skip: !envRes.data,
  });
  if (appRes.error) {
    throw appRes.error;
  }

  if (envRes.fetching || appRes.isLoading || appRes.isSkipped) {
    return null;
  }

  const { syncedFunctions } = appRes.data.latestSync ?? {};

  return (
    <div className="flex items-center justify-center bg-slate-100 pt-4">
      <div className="w-full max-w-[1200px]">
        <AppCard app={appRes.data} className="mb-4" />

        {appRes.data.latestSync && <GitCard className="mb-4" sync={appRes.data.latestSync} />}

        {syncedFunctions && <FunctionList envSlug={environmentSlug} functions={syncedFunctions} />}
      </div>
    </div>
  );
}
