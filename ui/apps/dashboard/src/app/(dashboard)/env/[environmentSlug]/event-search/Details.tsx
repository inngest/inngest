import { useState } from 'react';
import { EventDetails } from '@inngest/components/EventDetails';
import { classNames } from 'node_modules/@inngest/components/src/utils/classNames';

import LoadingIcon from '@/icons/LoadingIcon';
import { SlideOver } from './SlideOver';
import { useEvent } from './useEvent';

type Props = {
  envID: string;
  eventID: string | undefined;
  onClose: () => void;
};

export function Details({ envID, eventID, onClose }: Props) {
  const [selectedRunID, setSelectedRunID] = useState<string | undefined>(undefined);

  const isOpen = Boolean(eventID);

  const res = useEvent({ envID, eventID });
  if (res.error) {
    throw res.error;
  }

  let content;
  if (res.isLoading) {
    content = <Loading />;
  } else if (res.isSkipped) {
    content = null;
  } else {
    const { event, runs } = res.data;
    content = (
      <EventDetails
        event={event}
        functionRuns={runs}
        onFunctionRunClick={setSelectedRunID}
        selectedRunID={selectedRunID}
      />
    );
  }

  return (
    <SlideOver isOpen={isOpen} onClose={onClose} size={selectedRunID ? 'large' : 'small'}>
      <div
        className={classNames(
          'dark grid h-full text-white',
          selectedRunID ? 'grid-cols-2' : 'grid-cols-1'
        )}
      >
        {content}
      </div>
    </SlideOver>
  );
}

function Loading() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="flex flex-col items-center justify-center gap-2">
        <LoadingIcon />
        <div>Loading</div>
      </div>
    </div>
  );
}
