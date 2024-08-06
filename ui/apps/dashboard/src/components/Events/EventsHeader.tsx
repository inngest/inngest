import { Badge } from '@inngest/components/Badge';

import { Header } from '../Header/Header';
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
    breadcrumb={[{ text: 'Events', href: `/env/${envSlug}/events` }]}
    infoIcon={<EventInfo />}
    action={sendEvents && <SendEventButton newIANav={true} />}
    tabs={[
      {
        children: 'All events',
        href: `/env/${envSlug}/events`,
      },

      ...(eventSearch
        ? [
            {
              children: (
                <div className="flex flex-row items-center">
                  <div>Event Search</div>
                  <Badge
                    kind="solid"
                    className="text-warning border-warning ml-2 h-4 bg-amber-100 px-1.5 text-xs"
                  >
                    Experimental
                  </Badge>
                </div>
              ),
              href: `/env/${envSlug}/event-search`,
            },
          ]
        : []),
    ]}
  />
);
