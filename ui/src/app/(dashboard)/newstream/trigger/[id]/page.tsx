'use client';

import { useEffect } from 'react';
import { useParams } from 'next/navigation';

import { EventSection } from '@/components/Event/Section';
import { FunctionRunSection } from '@/components/Function/RunSection';
import { selectEvent } from '@/store/global';
import { useAppDispatch, useAppSelector } from '@/store/hooks';

export default function StreamDetails() {
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const dispatch = useAppDispatch();
  const params = useParams();

  useEffect(() => {
    if (params.id !== selectedEvent) {
      dispatch(selectEvent(params.id));
    }
  }, []);

  return (
    <>
      {selectedEvent && (
        <div className="grid grid-cols-2 h-full text-white">
          <EventSection eventId={params.id} />
          <FunctionRunSection runId={null} />
        </div>
      )}
    </>
  );
}
