'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { useClient } from 'urql';

import Input from '@/components/Forms/Input';
import {
  EventSearchFilterFieldDataType,
  EventSearchFilterOperator,
  type EventSearchFilterField,
} from '@/gql/graphql';
import { useEnvironment } from '@/queries';
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
  const [{ data: environment }] = useEnvironment({ environmentSlug });
  const client = useClient();

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFetching(true);

    try {
      const form = new FormData(event.currentTarget);
      const path = form.get('path');
      if (typeof path !== 'string') {
        // Should be unreachable
        throw new Error('path must be a string');
      }
      const value = form.get('value');
      if (typeof value !== 'string') {
        // Should be unreachable
        throw new Error('value must be a string');
      }
      if (!environment) {
        // Should be unreachable
        throw new Error('missing environment');
      }

      const fields: EventSearchFilterField[] = [
        {
          dataType: EventSearchFilterFieldDataType.Str,
          operator: EventSearchFilterOperator.Eq,
          path,
          value,
        },
      ];

      setEvents(
        await searchEvents({
          client,
          environmentID: environment.id,
          fields,
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
        <Input className="min-w-[300px]" type="text" name="path" placeholder="Path" required />
        <Input className="min-w-[300px]" type="text" name="value" placeholder="Value" required />
        <Button kind="primary" type="submit" disabled={fetching} label="Search" />
      </form>

      <EventTable environmentSlug={environmentSlug} events={events} />
    </>
  );
}
