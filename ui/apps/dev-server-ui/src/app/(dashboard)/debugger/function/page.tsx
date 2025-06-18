'use client';

import { Debugger } from '@inngest/components/Debugger/Debugger';
import { Error } from '@inngest/components/Error/Error';
import { Header } from '@inngest/components/Header/Header';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';

export default function Page() {
  const [functionSlug] = useSearchParam('function');

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }, { text: functionSlug ?? 'unknown' }, { text: 'Debug' }]}
        action={<div className="flex flex-row items-center gap-x-1"></div>}
      />
      {functionSlug ? (
        <Debugger functionSlug={functionSlug} />
      ) : (
        <Error message="Valid function required" />
      )}
    </>
  );
}
