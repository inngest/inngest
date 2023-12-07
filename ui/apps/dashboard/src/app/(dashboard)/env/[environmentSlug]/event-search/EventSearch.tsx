'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { IconInfo } from '@inngest/components/icons/Info';
import { useClient } from 'urql';

import Input from '@/components/Forms/Input';
import { useEnvironment } from '@/queries';
import { useSearchParam } from '@/utils/useSearchParam';
import { Details } from './Details';
import { EventTable } from './EventTable';
import { searchEvents } from './searchEvents';
import type { Event } from './types';

const day = 1000 * 60 * 60 * 24;

type Props = {
  environmentSlug: string;
};

export function EventSearch({ environmentSlug }: Props) {
  const [events, setEvents] = useState<Event[]>([]);
  const [fetching, setFetching] = useState(false);
  const [selectedEventID, setSelectedEventID] = useState<string | undefined>(undefined);
  const [{ data: environment }] = useEnvironment({ environmentSlug });
  const envID = environment?.id;
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
      if (!environment) {
        // Should be unreachable
        throw new Error('missing environment');
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
          <div className="flex items-start gap-1">
            <Button kind="primary" type="submit" loading={fetching} label="Search" />
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger>
                  <IconInfo className="h-4 w-4 text-slate-400" />
                </TooltipTrigger>
                <TooltipContent className="whitespace-pre-line">
                  Filter is only returning the last 3 days
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>
      </form>

      <EventTable
        events={events}
        onSelect={setSelectedEventID}
        blankState={getEmptyTableMessage({ fetching, query, events })}
      />

      {envID && (
        <Details
          envID={envID}
          eventID={selectedEventID}
          onClose={() => setSelectedEventID(undefined)}
          navigateToRun={() => <></>}
        />
      )}
    </>
  );
}
