'use client';

import { useRouter, useSearchParams } from 'next/navigation';

import SlideOver from '@/components/SlideOver';
import { selectEvent, selectRun } from '@/store/global';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import StreamDetails from '../StreamDetails';

const StreamSlideOver = () => {
  const params = useSearchParams();
  const triggerID = params.get('id');
  const router = useRouter();
  const dispatch = useAppDispatch();

  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);

  if (!selectedEvent && triggerID) {
    dispatch(selectEvent(triggerID));
  }

  const closeSlideOver = () => {
    router.push('/stream');
    dispatch(selectEvent(''));
    dispatch(selectRun(''));
  };

  if (!triggerID) return;

  return (
    <SlideOver onClose={closeSlideOver}>
      <StreamDetails />
    </SlideOver>
  );
};

export default StreamSlideOver;
