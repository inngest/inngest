'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { RiInformationLine } from '@remixicon/react';
import { useClient } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { Details } from './Details';
import { EventTable } from './EventTable';
import { searchEvents } from './searchEvents';
import type { Event } from './types';

const day = 1000 * 60 * 60 * 24;

export function EventSearch() {
  const [events, setEvents] = useState<Event[]>([]);
  const [fetching, setFetching] = useState(false);
  const [selectedEventID, setSelectedEventID] = useState<string | undefined>(undefined);
  const environment = useEnvironment();
  const client = useClient();

  const [query, setQuery] = useSearchParam('query');

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFetching(true);

    // TODO: Find a better way to extract form data.
    try {
      const form = new FormData(event.currentTarget);
      const newQuery = form.get('query');
      if (typeof newQuery !== 'string') {
        // Should be unreachable
        throw new Error('query must be a string');
      }

      setQuery(newQuery);

      setEvents(
        await searchEvents({
          client,
          environmentID: environment.id,
          query: newQuery,
          lowerTime: new Date(Date.now() - 3 * day),
          upperTime: new Date(),
        })
      );
    } finally {
      setFetching(false);
    }
  }

  function getEmptyTableMessage({
    fetching,
    query,
    events,
  }: {
    fetching: boolean;
    query?: string;
    events: Event[];
  }) {
    if (fetching) {
      return <p>Searching for events...</p>;
    } else if (query && events.length < 1) {
      return <p>No events found. Try adjusting your search.</p>;
    } else {
      return <p>Search to see events.</p>;
    }
  }

  return (
    <>
      <div className="block items-center justify-between lg:flex">
        <form onSubmit={onSubmit} className="m-4 flex gap-4">
          <div className="flex gap-4">
            <Input
              className="min-w-[800px]"
              defaultValue={query}
              name="query"
              placeholder="CEL query"
              required
              type="text"
            />
            <Button kind="primary" type="submit" loading={fetching} label="Search" />
          </div>
        </form>
        <div className="m-4 flex gap-1">
          <p className="text-sm">Experimental feature</p>
          <Tooltip>
            <TooltipTrigger>
              <RiInformationLine className="text-subtle h-4 w-4" />
            </TooltipTrigger>
            <TooltipContent className="whitespace-pre-line">
              This is an experimental feature, with a few limitations:
              <ul>
                <li> - Filter is only returning the last 3 days</li>
              </ul>
            </TooltipContent>
          </Tooltip>
        </div>
      </div>

      <EventTable
        events={events}
        onSelect={setSelectedEventID}
        blankState={getEmptyTableMessage({ fetching, query, events })}
      />

      <Details
        envID={environment.id}
        eventID={selectedEventID}
        onClose={() => setSelectedEventID(undefined)}
        navigateToRun={() => <></>}
      />
    </>
  );
}
