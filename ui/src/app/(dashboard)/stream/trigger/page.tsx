'use client';

import { useRouter, useSearchParams } from 'next/navigation';

import SlideOver from '@/components/SlideOver';
import StreamDetails from '../StreamDetails';

const StreamSlideOver = () => {
  const params = useSearchParams();
  const triggerID = params.get('event') || params.get('cron');
  const router = useRouter();

  const closeSlideOver = () => {
    router.push('/stream');
  };

  if (!triggerID) return;

  return (
    <SlideOver onClose={closeSlideOver}>
      <StreamDetails />
    </SlideOver>
  );
};

export default StreamSlideOver;
