'use client';

import { Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

import SlideOver from '@/components/SlideOver';
import StreamDetails from '../StreamDetails';

const StreamSlideOver = () => {
  const params = useSearchParams();
  const isEvent = params.get('event');
  const isCron = params.get('cron');
  const triggerID = isEvent || isCron;
  const router = useRouter();

  const closeSlideOver = () => {
    router.push('/stream');
  };

  if (!triggerID) return;

  return (
    <SlideOver size={isCron ? 'small' : 'large'} onClose={closeSlideOver}>
      <StreamDetails />
    </SlideOver>
  );
};

const StreamWrapper = () => {
  return (
    <Suspense>
      <StreamSlideOver />
    </Suspense>
  );
};

export default StreamWrapper;
