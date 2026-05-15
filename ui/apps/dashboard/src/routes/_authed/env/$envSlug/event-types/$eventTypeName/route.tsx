import { useState } from 'react';
import { CopyButton } from '@inngest/components/CopyButton';
import { Header } from '@inngest/components/Header/Header';
import { useCopyToClipboard } from '@inngest/components/hooks/useCopyToClipboard';
import { createFileRoute, Outlet } from '@tanstack/react-router';

import { ActionsMenu } from '@/components/Events/ActionsMenu';
import ArchiveEventModal from '@/components/Events/ArchiveEventModal';
import SendEventButton from '@/components/Events/SendEventButton';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute(
  '/_authed/env/$envSlug/event-types/$eventTypeName',
)({
  component: EventTypeLayout,
});

function EventTypeLayout() {
  const { envSlug, eventTypeName } = Route.useParams();
  const eventName = decodeURIComponent(eventTypeName);
  const [showArchive, setShowArchive] = useState(false);
  const { handleCopyClick, isCopying } = useCopyToClipboard();

  return (
    <>
      <Header
        breadcrumb={[
          { text: 'Event types', href: pathCreator.eventTypes({ envSlug }) },
          {
            text: eventName,
            href: pathCreator.eventType({ envSlug, eventName }),
          },
        ]}
        infoIcon={
          <CopyButton
            code={eventName}
            iconOnly
            size="small"
            isCopying={isCopying}
            handleCopyClick={handleCopyClick}
          />
        }
        tabs={[
          {
            href: pathCreator.eventType({ envSlug, eventName }),
            children: 'Dashboard',
            exactRouteMatch: true,
          },
          {
            children: 'Events',
            href: pathCreator.eventTypeEvents({ envSlug, eventName }),
          },
        ]}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            <ActionsMenu archive={() => setShowArchive(true)} />
            <SendEventButton eventName={eventName} />
          </div>
        }
      />

      <ArchiveEventModal
        eventName={eventName}
        isOpen={showArchive}
        onClose={() => {
          setShowArchive(false);
        }}
      />

      <Outlet />
    </>
  );
}
