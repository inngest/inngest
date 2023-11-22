'use client';

import { useCallback, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import type { NavigateToRunFn } from '@inngest/components/Timeline';
import { useClient } from 'urql';
import { z } from 'zod';

import Input from '@/components/Forms/Input';
import {
  EventSearchFilterFieldDataType,
  EventSearchFilterOperator,
  type EventSearchFilterField,
} from '@/gql/graphql';
import { useEnvironment } from '@/queries';
import { useSearchParam } from '@/utils/useSearchParam';
import { Details } from './Details';
import { EventTable } from './EventTable';
import { searchEvents } from './searchEvents';
import type { Event } from './types';

const day = 1000 * 60 * 60 * 24;

const fieldSchema = z.object({
  dataType: z.nativeEnum(EventSearchFilterFieldDataType),
  operator: z.nativeEnum(EventSearchFilterOperator),
  path: z.string(),
  value: z.string(),
});

type Props = {
  environmentSlug: string;
};

export function EventSearch({ environmentSlug }: Props) {
  const [events, setEvents] = useState<Event[]>([]);
  const [fetching, setFetching] = useState(false);
  const [selectedEventID, setSelectedEventID] = useState<string | undefined>(undefined);
  const [{ data: environment }] = useEnvironment({ environmentSlug });
  const envID = environment?.id;
  const envSlug = environment?.slug;
  const client = useClient();

  const [fields, setFields] = useFields();

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFetching(true);

    // TODO: Find a better way to extract form data.
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

      const newFields = [
        {
          dataType: EventSearchFilterFieldDataType.Str,
          operator: EventSearchFilterOperator.Eq,
          path,
          value,
        },
      ];

      setFields(newFields);

      setEvents(
        await searchEvents({
          client,
          environmentID: environment.id,
          fields: newFields,
          lowerTime: new Date(Date.now() - 3 * day),
          upperTime: new Date(),
        })
      );
    } finally {
      setFetching(false);
    }
  }

  const navigateToRun: NavigateToRunFn = useCallback(
    (opts) => {
      if (!environment?.slug) {
        return null;
      }

      return (
        <Link
          internalNavigation
          href={`/env/${encodeURIComponent(environment.slug)}/functions/${encodeURIComponent(
            opts.fnID
          )}/logs/${opts.runID}`}
        >
          Go to run
        </Link>
      );
    },
    [environment?.slug]
  );

  return (
    <>
      <form onSubmit={onSubmit} className="m-4 flex gap-4">
        {fields.map((field, index) => {
          return (
            // TODO: Don't use index as key.
            <div className="flex gap-4" key={index}>
              <Input
                className="min-w-[300px]"
                defaultValue={field.path}
                name="path"
                placeholder="Path"
                required
                type="text"
              />
              <Input
                className="min-w-[300px]"
                defaultValue={field.value}
                name="value"
                placeholder="Value"
                required
                type="text"
              />
              <Button kind="primary" type="submit" disabled={fetching} label="Search" />
            </div>
          );
        })}
      </form>

      <EventTable events={events} onSelect={setSelectedEventID} />

      {envID && (
        <Details
          envID={envID}
          eventID={selectedEventID}
          onClose={() => setSelectedEventID(undefined)}
          navigateToRun={navigateToRun}
        />
      )}
    </>
  );
}

function useFields(): [EventSearchFilterField[], (fields: EventSearchFilterField[]) => void] {
  const [fieldsParam, setFieldsParam] = useSearchParam('fields');

  const setFields = useCallback(
    (fields: EventSearchFilterField[]) => {
      setFieldsParam(JSON.stringify(fields));
    },
    [setFieldsParam]
  );

  let fields: EventSearchFilterField[] = [
    {
      dataType: EventSearchFilterFieldDataType.Str,
      operator: EventSearchFilterOperator.Eq,
      path: '',
      value: '',
    },
  ];
  if (fieldsParam) {
    fields = JSON.parse(fieldsParam).map((field: unknown) => {
      return fieldSchema.parse(field);
    });
  }

  return [fields, setFields];
}
