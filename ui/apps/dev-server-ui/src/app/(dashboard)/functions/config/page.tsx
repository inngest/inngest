'use client';

import { Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

import FunctionDetails from '@/app/(dashboard)/functions/config/FunctionDetails';
import SlideOver from '@/components/SlideOver';

const FunctionSlideOver = () => {
  const params = useSearchParams();
  const isEvent = params.get('slug');
  // const isCron = params.get('cron');
  // const triggerID = isEvent || isCron;
  const router = useRouter();

  const closeSlideOver = () => {
    console.log('closing');
    router.push('/functions');
  };

  if (!isEvent) return;

  return (
    <SlideOver size={true ? 'small' : 'large'} onClose={closeSlideOver}>
      <FunctionDetails />
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
