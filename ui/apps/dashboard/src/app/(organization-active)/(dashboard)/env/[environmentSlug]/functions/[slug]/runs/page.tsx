'use client';

import { use } from 'react';

import { Runs } from '@/components/Runs';

export default function Page(props: {
  params: Promise<{
    slug: string;
  }>;
}) {
  const params = use(props.params);
  const functionSlug = decodeURIComponent(params.slug);

  return <Runs functionSlug={functionSlug} scope="fn" />;
}
