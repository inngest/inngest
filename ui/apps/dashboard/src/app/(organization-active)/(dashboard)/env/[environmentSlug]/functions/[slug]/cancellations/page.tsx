'use client';

import { CancellationTable } from './CancellationTable';

type Props = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function Page({ params }: Props) {
  const envSlug = decodeURIComponent(params.environmentSlug);
  const fnSlug = decodeURIComponent(params.slug);

  return <CancellationTable envSlug={envSlug} fnSlug={fnSlug} />;
}
