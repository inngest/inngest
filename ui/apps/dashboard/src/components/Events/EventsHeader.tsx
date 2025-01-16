import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill';

import { EventInfo } from './EventInfo';
import SendEventButton from './SendEventButton';

export const EventsHeader = ({
  envSlug,
  eventSearch = false,
  sendEvents = false,
}: {
  envSlug: string;
  eventSearch?: boolean;
  sendEvents?: boolean;
}) => (
  <Header
    breadcrumb={[{ text: 'Events' }]}
    infoIcon={<EventInfo />}
    action={sendEvents && <SendEventButton />}
    tabs={[
      {
        children: 'All events',
        href: `/env/${envSlug}/events`,
      },

      ...(eventSearch
        ? [
            {
              children: (
                <div className="flex flex-row items-center gap-1">
                  <div>Event Search</div>
                  <Pill appearance="outlined" kind="warning">
                    Experimental
                  </Pill>
                </div>
              ),
              href: `/env/${envSlug}/event-search`,
            },
          ]
        : []),
    ]}
  />
);
