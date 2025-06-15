'use client';

import { Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

import SlideOver from '@/components/SlideOver';
import { FunctionConfigurationContainer } from './FunctionConfigurationContainer';

const FunctionSlideOver = () => {
  const params = useSearchParams();
  const functionSlug = params.get('slug');
  const router = useRouter();

  const closeSlideOver = () => {
    router.push('/functions');
  };

  if (!functionSlug) return;

  return (
    <SlideOver size="fixed-500" onClose={closeSlideOver}>
      <FunctionConfigurationContainer onClose={closeSlideOver} functionSlug={functionSlug} />
    </SlideOver>
  );
};

const FunctionWrapper = () => {
  return (
    <Suspense>
      <FunctionSlideOver />
    </Suspense>
  );
};

export default FunctionWrapper;
