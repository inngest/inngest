import { Button } from '@inngest/components/Button/Button';
import { RiArrowRightUpLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

type ExpandedRowProps = {
  eventName?: string;
  payload?: string;
  onReplay: (eventName: string, payload: string) => void;
  envSlug: string;
};

export function ExpandedRowActions({ eventName, payload, onReplay, envSlug }: ExpandedRowProps) {
  const isInternalEvent = eventName?.startsWith('inngest/');

  return (
    <div className="flex items-center gap-2">
      <Button
        label="Go to event type page"
        href={eventName ? pathCreator.eventType({ envSlug, eventName }) : undefined}
        appearance="ghost"
        size="small"
        icon={<RiArrowRightUpLine />}
        iconSide="left"
        disabled={!eventName}
      />
      <Button
        label="Replay event"
        onClick={() => eventName && payload && onReplay(eventName, payload)}
        appearance="outlined"
        size="small"
        disabled={!eventName || isInternalEvent || !payload}
      />
    </div>
  );
}
