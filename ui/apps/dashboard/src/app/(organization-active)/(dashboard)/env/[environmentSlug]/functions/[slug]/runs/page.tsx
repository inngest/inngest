'use client';

import { Runs } from '@/components/Runs';

export default function Page({
  params,
}: {
  params: {
    slug: string;
  };
}) {
  const functionSlug = decodeURIComponent(params.slug);

  return <Runs functionSlug={functionSlug} scope="fn" />;
}
