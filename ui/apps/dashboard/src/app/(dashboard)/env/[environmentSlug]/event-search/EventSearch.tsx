'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
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
          <Button kind="primary" type="submit" disabled={fetching} label="Search" />
        </div>
      </form>

      <EventTable events={events} onSelect={setSelectedEventID} />

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
