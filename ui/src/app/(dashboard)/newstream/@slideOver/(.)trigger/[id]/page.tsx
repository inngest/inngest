'use client';

import { useRouter } from 'next/navigation';
import SlideOver from '@/components/SlideOver';
import StreamDetails from '../../StreamDetails';
import { selectEvent, selectRun } from '@/store/global';
import { useAppDispatch } from '@/store/hooks';

export default function Page() {
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
