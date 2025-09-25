'use client';

import { use } from 'react';

import { CancellationTable } from './CancellationTable';

type Props = {
  params: Promise<{
    environmentSlug: string;
    slug: string;
  }>;
};

export default function Page(props: Props) {
  const params = use(props.params);
  const envSlug = decodeURIComponent(params.environmentSlug);
  const fnSlug = decodeURIComponent(params.slug);

  return <CancellationTable envSlug={envSlug} fnSlug={fnSlug} />;
}
