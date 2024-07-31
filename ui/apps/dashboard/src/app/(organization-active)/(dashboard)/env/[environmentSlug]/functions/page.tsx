import type { ReactNode } from 'react';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { FunctionList } from '@/components/Functions/FunctionsList';
import { Header } from '@/components/Header/Header';
import { FunctionsHeader } from './oldHeader';

type FunctionLayoutProps = {
  params: {
    environmentSlug: string;
    archived?: string;
  };
};

export default async function FunctionPage({
  params: { environmentSlug, archived: archivedParam },
}: FunctionLayoutProps) {
  const newIANav = await getBooleanFlag('new-ia-nav');
  const archived = archivedParam === 'true';

  return (
    <>
      {newIANav ? (
        <Header breadcrumb={[{ text: 'Functions', href: `/env/${environmentSlug}/functions` }]} />
      ) : (
        <FunctionsHeader />
      )}
      <FunctionList envSlug={environmentSlug} archived={archived} />
    </>
  );
}
