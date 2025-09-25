'use client';

import { use } from 'react';
import { Debugger } from '@inngest/components/Debugger/Debugger';
import { Header } from '@inngest/components/Header/Header';

export default function Page(props: { params: Promise<{ functionSlug: string }> }) {
  const params = use(props.params);
  const functionSlug = decodeURIComponent(params.functionSlug);

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }, { text: functionSlug }, { text: 'Debug' }]}
        action={<div className="flex flex-row items-center gap-x-1"></div>}
      />
      <Debugger functionSlug={functionSlug} />
    </>
  );
}
