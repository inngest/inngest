'use client';

import { useCallback, useEffect, useState } from 'react';
import { useUser } from '@clerk/nextjs';

export type Schemas = Record<string, unknown>;
export type EventTypes = string[];

export function useEvents() {
  const { user } = useUser();

  const fetchSchemas = useCallback(async (signal?: AbortSignal): Promise<Schemas> => {
    const response = await fetch('/api/schemas', { method: 'GET', signal });
    if (!response.ok) {
      throw new Error(`Failed to fetch schemas: ${response.status}`);
    }
    return (await response.json()) as Schemas;
  }, []);

  const fetchEventTypes = useCallback(async (signal?: AbortSignal): Promise<EventTypes> => {
    const response = await fetch('/api/events', { method: 'GET', signal });
    if (!response.ok) {
      throw new Error(`Failed to fetch event types: ${response.status}`);
    }
    return (await response.json()) as EventTypes;
  }, []);

  const [schemas, setSchemas] = useState<Schemas | null>(null);
  const [eventTypes, setEventTypes] = useState<EventTypes>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!user?.id) {
      setSchemas(null);
      setEventTypes([]);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    (async () => {
      try {
        const [ev, sch] = await Promise.all([
          fetchEventTypes(controller.signal),
          fetchSchemas(controller.signal),
        ]);
        setEventTypes(ev);
        setSchemas(sch);
      } catch (err) {
        if ((err as { name?: string } | undefined)?.name !== 'AbortError') {
          setError(err instanceof Error ? err : new Error('Failed to fetch event data'));
        }
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    })();
    return () => controller.abort();
  }, [user?.id, fetchEventTypes, fetchSchemas]);

  return { schemas, eventTypes, loading, error, fetchSchemas, fetchEventTypes };
}
