'use client';

import { useRouter } from 'next/navigation';

import SlideOver from '@/components/SlideOver';
import { selectEvent, selectRun } from '@/store/global';
import { useAppDispatch } from '@/store/hooks';
import StreamDetails from '../../StreamDetails';

export default function SlideOverStreamDetailsPage() {
  const router = useRouter();
  const dispatch = useAppDispatch();

  function handleCloseSlideOver() {
    router.back();
    dispatch(selectEvent(''));
    dispatch(selectRun(''));
  }

  return (
    <SlideOver onClose={handleCloseSlideOver}>
      <StreamDetails />
    </SlideOver>
  );
}
