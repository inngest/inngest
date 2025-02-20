'use client';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { Runs } from '@/components/Runs';

export default function Page({
  params,
}: {
  params: {
    slug: string;
  };
}) {
  const functionSlug = decodeURIComponent(params.slug);
  const { value: traceAIEnabled, isReady: featureFlagReady } = useBooleanFlag('ai-traces');

  return (
    <Runs
      functionSlug={functionSlug}
      scope="fn"
      traceAIEnabled={featureFlagReady && traceAIEnabled}
    />
  );
}
