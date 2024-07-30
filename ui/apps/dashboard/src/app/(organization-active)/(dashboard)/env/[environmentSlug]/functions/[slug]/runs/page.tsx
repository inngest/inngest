'use client';

import { FnRuns } from '@/components/Runs/FnRuns';

export default function Page({
  params,
}: {
  params: {
    slug: string;
  };
}) {
  const functionSlug = decodeURIComponent(params.slug);

  return <FnRuns functionSlug={functionSlug} />;
}
